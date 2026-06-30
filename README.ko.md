# AIBrow

AIBrow은 Go로 작성된 AI 기반 사진 편집기입니다. FFmpeg 기반 LCH 색상 분석과 LLM 기반 도구 오케스트레이션을 결합하여 REST API와 웹 UI를 제공합니다.

## 주요 기능

- 이미지 업로드 및 원본/편집/변환 파일 관리
- FFmpeg를 이용한 LCH 색 공간(명도, 채도, 색상) 분석
- 밝기, 대비, 채도, 감마, 마이크로 콘트라스트, 색온도 조정
- `png`, `jpg`, `webp`, `tiff`, `bmp`, `avif` 형식 변환
- `POST /api/v1/chat`으로 LLM과 대화하며 이미지 도구 호출
- `POST /api/v1/request`로 자연어 지시어로 사진 편집 요청
- `POST /api/v1/images/{id}/export`로 분석/조정 프리셋 Excel 내보내기
- OpenAI, Google Gemini, Ollama를 지원하는 공급자 중립 LLM 레이어
- Go 구문 커버리지 100% 유지

## 요구 사항

- Go 1.26.4 이상
- FFmpeg 8.x 이상(호환 버전)
- LLM 기능을 사용하려면 아래 중 하나:
  - `OPENAI_API_KEY`
  - `GEMINI_API_KEY`
  - 로컬 Ollama 서버(기본: `http://localhost:11434`)

## 빠른 시작

```bash
# 의존성 설치
go mod tidy

# 기본 설정으로 실행
go run ./cmd/aibrow
```

REST 서버는 `http://localhost:8080`, Swagger UI는 `http://localhost:8081`에서 시작됩니다.

## 설정

`configs/.env`를 만들거나 환경 변수를 설정하세요.

| 변수 | 기본값 | 설명 |
|------|--------|------|
| `REST_PORT` | `8080` | REST API 포트 |
| `SWAGGER_PORT` | `8081` | Swagger UI 포트 |
| `UPLOAD_DIR` | `uploads` | 업로드된 이미지 저장 위치 |
| `OUTPUT_DIR` | `outputs` | 편집/변환/내보내기 파일 저장 위치 |
| `LOG_PATH` | `logs/error.log` | 애플리케이션 로그 파일 |
| `OPENAI_API_KEY` | - | OpenAI API 키 |
| `OPENAI_BASE_URL` | - | OpenAI 호환 베이스 URL |
| `OPENAI_MODEL` | `gpt-4o` | OpenAI 모델 |
| `GEMINI_API_KEY` | - | Gemini API 키 |
| `GEMINI_MODEL` | `gemini-1.5-flash` | Gemini 모델 |
| `OLLAMA_BASE_URL` | `http://localhost:11434` | Ollama 서버 URL |
| `OLLAMA_MODEL` | `llama3.1` | Ollama 모델 |

별도 공급자를 지정하지 않으면 우선순위는 OpenAI → Gemini → Ollama입니다.

## API 개요

| 메서드 | 경로 | 설명 |
|--------|------|------|
| GET | `/health` | 헬스 체크 |
| POST | `/api/v1/images/upload` | 이미지 업로드 |
| GET | `/api/v1/images/{id}` | 원본 이미지 다운로드 |
| POST | `/api/v1/images/{id}/analyze` | LCH 분석 실행 |
| POST | `/api/v1/images/{id}/adjust` | 조정 적용 |
| POST | `/api/v1/images/{id}/convert` | 형식 변환 |
| GET | `/api/v1/images/{id}/processed` | 편집된 이미지 다운로드 |
| POST | `/api/v1/images/{id}/export` | 프리셋 Excel 내보내기 |
| POST | `/api/v1/chat` | 도구 호출이 가능한 LLM 채팅 |
| POST | `/api/v1/request` | 자연어 편집 요청 |
| GET | `/swagger/index.html` | Swagger UI |

요청/응답 스키마는 Swagger를 참조하세요.

## 아키텍처

```
cmd/aibrow          메인 진입점
internal/app        부트스트랩, 포트 선택, 서버 연결
internal/config     환경 설정 로드
internal/logging    파일/표준출력 이중 로거
internal/ports      사용 가능 포트 스캐너
src/chat            도구 실행 루프를 포함한 LLM 채팅 오케스트레이션
src/ffmpeg          FFmpeg 클라이언트, LCH 분석, 조정 필터
src/llm             공급자 중립 LLM 인터페이스
src/llm/factory     환경 기반 공급자 선택
src/llm/openai      OpenAI 호환 드라이버
src/llm/gemini      Google Gemini 드라이버
src/llm/ollama      Ollama 드라이버
src/server          Gin 서버, REST 핸들러, Swagger 연결
src/store           인메모리 이미지 레코드 저장소
src/tools           MCP 스타일 도구 레지스트리(분석, 조정, 변환, 내보내기)
web                 정적 프론트엔드
```

## 테스트

```bash
go test ./...
```

모든 패키지는 100% 구문 커버리지를 유지합니다. 확인하려면:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## 벤치마크

로컬 개발 환경(Linux arm64)에서 실행:

```bash
go test -bench=. -benchmem ./src/ffmpeg
```

예시 결과:

```text
BenchmarkAnalyze-8    6    248185077 ns/op    4462868 B/op    254 allocs/op
BenchmarkAdjust-8     8    183869431 ns/op     18024 B/op     99 allocs/op
```

## 라이선스

MIT
