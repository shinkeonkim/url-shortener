# URL Shortener 구현 기획

## 목표

`*.url.shinkeonkim.com`의 서브도메인을 짧은 키로 사용해 URL을 생성·조회하고,
관리자만 매핑을 변경하며, 클릭 통계와 QR 코드를 확인할 수 있는 Go 서비스를 만든다.
SQLite 단일 인스턴스로 시작하고 Docker/Kubernetes, CI/CD, Prometheus/Grafana 연동까지
재현 가능한 형태로 제공한다.

## 주요 결정

- Go 표준 `net/http` 라우팅과 명시적 계층(`config`, `store`, `service`, `http`)을 사용한다.
- 공개 요청은 `{slug}.url.shinkeonkim.com`을 원본 URL로 리디렉션한다. 로컬·API 사용을
  위해 `/r/{slug}`도 동일하게 지원한다.
- 관리 API와 UI는 환경변수로 주입한 단일 관리자 계정, bcrypt 비밀번호, 서명된
  HttpOnly 세션 쿠키로 보호한다. 변경 API는 세션 또는 Bearer 관리자 토큰을 받는다.
- SQLite는 WAL, foreign key, busy timeout을 켜고 스키마 버전을 애플리케이션 시작 때
  적용한다. 삭제는 관련 클릭 이벤트와 함께 원자적으로 처리한다.
- 통계는 총 클릭 수와 최근 클릭 이벤트(시각, referrer, user-agent)를 제공한다.
  IP 주소는 저장하지 않아 불필요한 개인정보 수집을 피한다.
- `/health`는 접근 로그에서 제외하고, `/metrics`는 Prometheus 형식으로 제공한다.
- 운영 이미지는 non-root, read-only root filesystem을 전제로 하고 SQLite만 PVC에 둔다.
- Kubernetes 리소스는 Kustomize base/production overlay로 관리한다. wildcard DNS/TLS는
  홈랩에서 사전 준비하며 서비스 저장소에는 평문 Secret을 커밋하지 않는다.

## API 계약

| Method | Path | 권한 | 설명 |
| --- | --- | --- | --- |
| GET | `/health` | 공개 | liveness/readiness 상태 |
| GET | `/r/{slug}` | 공개 | 원본 URL로 302 리디렉션 |
| GET | `/api/v1/urls/{slug}` | 공개 | 매핑 기본 정보 조회 |
| GET | `/api/v1/urls/{slug}/qr` | 공개 | short URL QR PNG |
| POST | `/api/v1/auth/login` | 공개 | 관리자 세션 생성 |
| POST | `/api/v1/auth/logout` | 관리자 | 관리자 세션 폐기 |
| POST | `/api/v1/urls` | 관리자 | 사용자 지정 slug 매핑 생성 |
| DELETE | `/api/v1/urls/{slug}` | 관리자 | 매핑 삭제 |
| GET | `/api/v1/urls/{slug}/stats` | 관리자 | 클릭 통계 조회 |
| GET | `/metrics` | 클러스터 | Prometheus 메트릭 |

## 작업 스택

1. 프로젝트 기반, 설정, SQLite 저장소, health endpoint와 단위 테스트
2. 공개 URL API, 서브도메인/경로 리디렉션, 클릭 통계, QR 코드
3. 관리자 인증, URL 생성·삭제·통계 API, 반응형 관리 UI
4. 구조화 로그, Prometheus 메트릭, Docker 이미지, CI 및 통합 테스트
5. Kubernetes/Kustomize, GitHub Actions CD, Grafana 대시보드, 운영 문서와 E2E

각 단계는 직전 브랜치를 base로 하는 stacked pull request로 만들며, 개별 PR은 자체
테스트를 통과하고 다음 단계가 앞 PR의 계약에만 의존하도록 한다.

## 완료 기준

- `go test ./...`, race test, 통합/E2E 테스트가 통과한다.
- 모든 Go 소스 파일은 200줄 이하이며 패키지 책임이 분리돼 있다.
- 인증 없이 생성·삭제·통계를 호출하면 거부된다.
- redirect가 클릭을 기록하고 통계/Prometheus 지표에 반영된다.
- health 요청은 일반 access log에 남지 않는다.
- 컨테이너 및 Kubernetes 매니페스트가 로컬 정적 검증을 통과한다.
- README에 로컬 실행, 설정, API, Docker, Kubernetes, 기여 및 운영 절차가 있다.

## 범위 밖

- 다중 사용자 가입, 소셜 로그인, 과금, 분산 SQLite, 브라우저 fingerprinting
- DNS provider 레코드와 홈랩 ArgoCD Application의 직접 변경(별도 GitOps 저장소 작업)
