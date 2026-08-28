# URL Shortener

Go와 SQLite로 만든 self-hosted URL shortener입니다. `demo.url.shinkeonkim.com` 같은
wildcard subdomain 또는 `/r/demo`를 원본 URL로 리디렉션하고 클릭 통계와 QR 코드를
제공합니다. 생성·삭제·상세 통계는 관리자만 사용할 수 있습니다.

## 로컬 실행

요구 사항은 Go 1.25+입니다.

```bash
export ADMIN_USERNAME=admin
export ADMIN_PASSWORD_HASH="$(htpasswd -bnBC 12 '' 'change-me' | tr -d ':\n')"
export ADMIN_TOKEN="$(openssl rand -hex 32)"
export SESSION_KEY="$(openssl rand -hex 32)"
export COOKIE_SECURE=false
go run ./cmd/url-shortener
```

관리 UI는 `http://localhost:8080/admin`, health는 `/health`, 메트릭은 `/metrics`입니다.
기본 DB 파일은 현재 디렉터리의 `url-shortener.db`이며 `DATABASE_PATH`로 바꿀 수 있습니다.

## 설정

| 환경변수 | 기본값 | 설명 |
| --- | --- | --- |
| `ADDRESS` | `:8080` | HTTP listen address |
| `DATABASE_PATH` | `url-shortener.db` | SQLite 파일 위치 |
| `BASE_DOMAIN` | `url.shinkeonkim.com` | slug를 붙일 wildcard domain |
| `ADMIN_USERNAME` | 없음 | 관리자 로그인 이름 |
| `ADMIN_PASSWORD_HASH` | 없음 | bcrypt 해시(평문 금지) |
| `ADMIN_TOKEN` | 없음 | 자동화용 Bearer token |
| `SESSION_KEY` | 없음 | 세션 HMAC key(32 random bytes 이상 권장) |
| `COOKIE_SECURE` | `true` | HTTPS에서만 cookie 전송 |

## API 예시

```bash
curl -H "Authorization: Bearer $ADMIN_TOKEN" -H 'Content-Type: application/json' \
  -d '{"slug":"go","target_url":"https://go.dev"}' http://localhost:8080/api/v1/urls
curl -I http://localhost:8080/r/go
curl http://localhost:8080/api/v1/urls/go/qr -o go.png
curl -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:8080/api/v1/urls/go/stats
curl -X DELETE -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:8080/api/v1/urls/go
```

slug는 DNS label에 안전한 소문자, 숫자, 하이픈 1–63자로 제한됩니다. 대상은 사용자 정보가
없는 절대 `http`/`https` URL만 허용합니다. 클릭 이벤트에는 IP를 저장하지 않습니다.

## 테스트와 컨테이너

```bash
make check
go test -race ./...
make e2e
docker build -t url-shortener .
docker run --rm -p 8080:8080 -v url-data:/data --env-file .env url-shortener
```

CI는 formatting, 200줄 제한, vet, race/coverage, E2E와 container build를 검사합니다.
main과 `v*` tag는 이미지를 `ghcr.io/shinkeonkim/url-shortener`에 게시합니다.

## Kubernetes

```bash
kubectl apply -f deploy/secret.example.yaml # 먼저 값을 교체하거나 SOPS로 관리
kubectl apply -k deploy/overlays/production
```

production overlay는 단일 replica/Recreate deployment, Longhorn RWO PVC, Traefik wildcard
Ingress, cert-manager Certificate, ServiceMonitor, Grafana dashboard와 alerts를 포함합니다.
클릭 원본 이벤트는 30일 뒤 일별 통계로, 1년 뒤 월별 통계로 자동 축약되며 전체 클릭
수는 계속 보존됩니다.
SQLite 때문에 replica를 1보다 높이면 안 됩니다. 자세한 준비·검증·롤백은
[`docs/operations.md`](docs/operations.md)를 참고하세요.

## 문서와 기여

- 요구사항: [`REQUEST.md`](REQUEST.md)
- 구현 기획과 API 계약: [`PLAN.md`](PLAN.md)
- 운영: [`docs/operations.md`](docs/operations.md)
- 기여 방법: [`CONTRIBUTING.md`](CONTRIBUTING.md)

이 프로젝트는 오픈 소스로 제공됩니다. 라이선스를 확정하기 전까지는 재배포 조건을
저장소 소유자에게 확인해 주세요.
