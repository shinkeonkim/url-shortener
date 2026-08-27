# Contributing

작은 단위의 issue와 pull request를 선호합니다. 변경 전 issue에 목적과 완료 조건을 적고,
의존 작업이 있으면 PR base를 직전 작업 branch로 지정해 stacked PR로 제출하세요.

```bash
make check
go test -race ./...
make e2e
```

Go 파일은 200줄 이하로 유지하고 패키지의 책임을 늘리기보다 새 파일/타입으로 분리합니다.
공개 API 변경에는 HTTP 테스트를, 저장소 변경에는 재시작을 포함한 테스트를 추가합니다.
비밀번호, token, kubeconfig, 실제 Secret은 commit하지 마세요. 보안 문제는 공개 issue 대신
저장소 소유자에게 비공개로 알려 주세요.
