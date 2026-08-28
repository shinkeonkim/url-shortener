# 배포 및 운영 가이드

## 사전 준비

1. `*.url.shinkeonkim.com` wildcard DNS를 홈랩 Traefik의 공개 진입점으로 연결합니다.
2. `cloudflare-dns01` ClusterIssuer가 `shinkeonkim.com` zone의 DNS-01 challenge를 수행할
   수 있는지 확인합니다. 현재 provider가 다르면 `certificate.yaml`의 issuer를 바꿉니다.
3. `url-shortener-secrets`를 SOPS/ksops 등 홈랩의 비밀 관리 흐름으로 생성합니다.
   `deploy/secret.example.yaml`은 형태만 제공하며 `CHANGEME` 값을 적용하면 안 됩니다.
4. GHCR package를 public으로 두거나 namespace에 `imagePullSecret`을 연결합니다.
5. oh-my-homelab ArgoCD에 이 저장소의 `deploy/overlays/production`을 가리키는
   Application을 별도 PR로 등록합니다. 서비스 저장소는 홈랩 저장소를 직접 변경하지 않습니다.

## 검증

```bash
kubectl kustomize deploy/overlays/production >/dev/null
kubectl -n url-shortener rollout status deploy/url-shortener
kubectl -n url-shortener get certificate,pvc,pod
kubectl -n url-shortener port-forward svc/url-shortener 18080:8080
curl -fsS http://127.0.0.1:18080/health
curl -fsS http://127.0.0.1:18080/metrics | grep url_shortener
```

외부에서는 임시 slug를 만들어 `https://<slug>.url.shinkeonkim.com`의 TLS, redirect,
통계 증가를 차례로 확인하고 삭제합니다. Grafana에서 `URL Shortener` dashboard와 Loki
`{namespace="url-shortener"}` JSON 로그를 확인합니다. `/health`는 access log에 없어야 합니다.

## 백업과 복구

SQLite WAL을 포함한 일관된 백업은 실행 중인 pod에서 `.backup` 명령을 쓰거나 배포를
내린 뒤 PVC snapshot을 만듭니다. DB 파일만 실행 중에 복사하지 않습니다. Longhorn의
정기 snapshot/backup을 설정하고 분기마다 별도 namespace에서 복구 훈련을 수행합니다.

## 로그 및 통계 보존

애플리케이션은 매 시작 시와 24시간마다 통계 compaction을 수행합니다. 최근 30일 클릭은
referrer와 user-agent를 포함한 원본 이벤트로, 30일 이후 1년까지는 일별 클릭 수로,
그 이후에는 월별 클릭 수로 영구 보존합니다. `/api/v1/urls/{slug}/stats`는 최근 이벤트와
집계 구간을 함께 반환합니다. Loki access log는 홈랩 보존 정책에 따라 30일 뒤 삭제하며,
장기 추이는 SQLite rollup과 Prometheus 지표를 사용합니다.

## 배포와 롤백

main merge 후 GHCR `latest`와 commit SHA tag가 게시됩니다. GitOps에서 운영 image를
검증한 SHA tag로 고정하는 것을 권장합니다. 롤백은 이전 정상 SHA로 deployment image를
되돌린 뒤 ArgoCD manifest에도 같은 값을 반영합니다. schema는 현재 additive이므로 이전
binary가 읽을 수 있지만, 향후 destructive migration은 별도 백업/복구 계획이 필요합니다.

장애 시 `rollout status`, pod events, JSON logs, PVC 상태 순으로 확인합니다. SQLite lock가
지속되면 replica가 정확히 1인지와 PVC가 RWO인지 먼저 검사합니다.
