# 개인 미니 프로젝트 - URL 단축 서비스

### 사용 기술
- Go
- PostgresDB

### /shorten - POST
- 입력받은 URL을 8자리 ID로 축소시켜 응답을 보냅니다.

#### INPUT
```json
{
  "url": "https://github.com/Chemuchi"
}
```

#### OUTPUT
```json
{
  "short_url": "Z6wzIpDT"
}
```
### /{short_url} - GET
- 응답받은 8자리 ID 를 입력하면 리다이렉트 시켜줍니다.


### TODO
- 로그에 IP, UA 출력 ✅
- 로그를 DB에도 저장
- 오래된 URL과 ID는 자동으로 삭제
- 같은 URL은 새로 제작하지 않고 이미 존재하는 ID를 찾아 출력
- 프론트페이지 제작
- Github Actions 로 CI/CD 구현