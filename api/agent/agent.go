package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PedagogicalAgent struct {
	classifier      *IntentClassifier
	contextMgr      *ContextManager
	decisionEngine  *DecisionEngine
	planner         *Planner
	toolRegistry    *ToolRegistry
	responseGen     *ResponseGenerator
	learningUpdater *LearningUpdater
	auditLogger     *AuditLogger
	cfg             *AgentConfig
}

type AgentRequest struct {
	SessionID string `json:"session_id"`
	StudentID string `json:"student_id"`
	CourseID  string `json:"course_id"`
	Message   string `json:"message"`
	Mode      string `json:"mode"`
}

type AgentResponse struct {
	SessionID string              `json:"session_id"`
	Intent    IntentType          `json:"intent"`
	Strategy  PedagogicalStrategy `json:"strategy,omitempty"`
	Response  string              `json:"response"`
	Citations []map[string]any    `json:"citations,omitempty"`
	Actions   []string            `json:"actions,omitempty"`
	Sections  map[string]string   `json:"sections,omitempty"`
}

func NewPedagogicalAgent(
	db *pgxpool.Pool,
	cfg *AgentConfig,
	toolRegistry *ToolRegistry,
	callLLM LLMFunc,
) *PedagogicalAgent {
	return &PedagogicalAgent{
		classifier:      NewIntentClassifier(db, cfg, callLLM),
		contextMgr:      NewContextManager(db),
		decisionEngine:  NewDecisionEngine(cfg),
		planner:         NewPlanner(NewDecisionEngine(cfg)),
		toolRegistry:    toolRegistry,
		responseGen:     NewResponseGenerator(callLLM, cfg),
		learningUpdater: NewLearningUpdater(db),
		auditLogger:     NewAuditLogger(db),
		cfg:             cfg,
	}
}

func (pa *PedagogicalAgent) Process(ctx context.Context, req *AgentRequest) (*AgentResponse, error) {
	start := time.Now()

	trace := &AgentTrace{
		StudentID: req.StudentID,
		SessionID: req.SessionID,
		CreatedAt: time.Now(),
	}
	defer func() {
		trace.Duration = time.Since(start)
		go pa.auditLogger.LogExecution(context.Background(), trace)
	}()

	citationMgr := NewCitationManager()

	studentCtx, err := pa.contextMgr.LoadStudentContext(ctx, req.StudentID, req.CourseID)
	if err != nil {
		studentCtx = &StudentContext{StudentID: req.StudentID, CourseID: req.CourseID}
	}
	studentCtx.CurrentMode = req.Mode

	state := &AgentState{
		SessionID: req.SessionID,
		StudentID: req.StudentID,
		CourseID:  req.CourseID,
		Mode:      req.Mode,
	}

	frustrationDetected := pa.classifier.DetectFrustration(req.Message)

	multiIntent, _ := pa.classifier.ClassifyMulti(ctx, req.Message, state.Messages)

	var primaryIntent IntentType
	var primaryConcept string
	if multiIntent != nil && len(multiIntent.Intents) > 0 {
		primaryIntent = multiIntent.Intents[0].Intent
		primaryConcept = multiIntent.Intents[0].Concept
	} else {
		singleIntent, err := pa.classifier.Classify(ctx, req.Message, state.Messages)
		if err != nil || singleIntent == nil {
			singleIntent = &IntentResult{Intent: IntentAskTheory, Confidence: 0.5}
		}
		primaryIntent = singleIntent.Intent
		primaryConcept = singleIntent.Concept
	}

	state.Intent = primaryIntent
	if primaryConcept != "" {
		state.CurrentConcept = primaryConcept
		studentCtx.CurrentTopic = primaryConcept
	}

	plan := pa.planner.CreatePlan(ctx, primaryIntent, studentCtx, req.Message, frustrationDetected)
	trace.Intent = plan.Intent
	trace.Plan = plan

	actions := []string{string(primaryIntent)}
	if multiIntent != nil {
		for _, mi := range multiIntent.Intents {
			if mi.Intent != primaryIntent {
				actions = append(actions, string(mi.Intent))
			}
		}
	}

	toolResults := make([]*ToolCall, 0, len(plan.Steps))
	toolCalls := 0

	for _, step := range plan.Steps {
		if toolCalls >= pa.cfg.MaxToolCalls {
			break
		}

		tc := &ToolCall{
			Tool:    step.Tool,
			Purpose: step.Purpose,
			Input:   step.Input,
		}

		err := pa.toolRegistry.Execute(ctx, tc)
		toolResults = append(toolResults, tc)
		toolCalls++

		if err != nil {
			tc.Error = err.Error()
		}

		if step.Tool == "rag_search" && tc.Result != nil {
			if sourcesRaw, ok := tc.Result["sources"]; ok {
				if sources, ok := sourcesRaw.([]any); ok {
					wrapper := []map[string]any{{"sources": sources}}
					citationMgr.AddFromToolResults(wrapper)
				}
			}
		}
	}

	trace.ToolResults = toolResults

	agentResp, err := pa.responseGen.Generate(ctx, plan, toolResults, req.Message, citationMgr)
	if err != nil {
		agentResp = pa.responseGen.fallbackResponse(req.Message, toolResults, citationMgr, plan)
	}

	trace.Response = agentResp.Response

	if primaryConcept != "" {
		err := pa.learningUpdater.UpdateAfterInteraction(ctx, req.StudentID, req.CourseID, primaryConcept, false, 0, 0)
		if err != nil {
			_ = err
		}
	}

	resp := &AgentResponse{
		SessionID: req.SessionID,
		Intent:    primaryIntent,
		Strategy:  plan.Strategy,
		Response:  agentResp.Response,
		Citations: agentResp.Citations,
		Actions:   actions,
	}

	return resp, nil
}

func (pa *PedagogicalAgent) GetToolRegistry() *ToolRegistry {
	return pa.toolRegistry
}

func (pa *PedagogicalAgent) GetAgentConfig() *AgentConfig {
	return pa.cfg
}

func (pa *PedagogicalAgent) DefaultAgentConfig() *AgentConfig {
	cfg := DefaultAgentConfig()
	return &cfg
}

type PedagogicalAgentHTTP struct {
	Agent *PedagogicalAgent
	DB    *pgxpool.Pool
}

func NewPedagogicalAgentHTTP(pa *PedagogicalAgent, db *pgxpool.Pool) *PedagogicalAgentHTTP {
	return &PedagogicalAgentHTTP{Agent: pa, DB: db}
}

func (h *PedagogicalAgentHTTP) ProcessRequest(ctx context.Context, req *AgentRequest) (*AgentResponse, error) {
	if req.SessionID == "" {
		req.SessionID = fmt.Sprintf("agent-%d", time.Now().UnixNano())
	}
	if req.CourseID == "" {
		req.CourseID = "matematica-1"
	}
	if req.Mode == "" {
		req.Mode = "tutor"
	}

	return h.Agent.Process(ctx, req)
}
