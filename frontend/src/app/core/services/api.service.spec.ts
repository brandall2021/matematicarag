import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ApiService } from './api.service';
import { environment } from '../../../environments/environment';

describe('ApiService.agentChat', () => {
  let service: ApiService;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [ApiService, provideHttpClient(), provideHttpClientTesting()],
    });
    service = TestBed.inject(ApiService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
  });

  it('posts the message under the "message" field the backend expects', () => {
    service.agentChat('Explicame como derivar x^2 + 3x', 'sess-1', 'curso-1').subscribe();

    const req = httpMock.expectOne(`${environment.apiUrl}/api/agent/chat`);
    expect(req.request.method).toBe('POST');
    const body = req.request.body;
    expect(body['message']).toBe('Explicame como derivar x^2 + 3x');
    expect(body['query']).toBeUndefined();
    expect(body['session_id']).toBe('sess-1');
    expect(body['course_id']).toBe('curso-1');

    req.flush({});
  });
});
