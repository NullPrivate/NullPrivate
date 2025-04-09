# AdGuardPrivate

AdGuardPrivate은 _AdGuardHome_의 포크로, 향상된 기능과 사용자 정의 가능성을 제공하는 SaaS 호스팅 버전을 제공하도록 설계되었습니다. [AdGuard Private](https://adguardprivate.com)에서 호스팅됩니다.

## 주요 기능

### 원래 기능

1. **네트워크 전반에 걸친 광고 차단**

   - 네트워크 내 모든 장치에서 광고 및 추적기를 차단합니다.
   - 추적 도메인을 "블랙홀"로 재라우팅하는 DNS 서버로 작동합니다.

2. **사용자 정의 필터링 규칙**

   - 자신만의 사용자 정의 필터링 규칙을 추가합니다.
   - 네트워크 활동을 모니터링하고 제어합니다.

3. **암호화된 DNS 지원**

   - DNS-over-HTTPS, DNS-over-TLS 및 DNSCrypt를 지원합니다.

4. **내장 DHCP 서버**

   - 기본적으로 DHCP 서버 기능을 제공합니다.

5. **클라이언트별 설정**

   - 개별 장치에 대한 설정을 구성합니다.

6. **부모 통제**

   - 성인 도메인을 차단하고 검색 엔진에서 안전 검색을 적용합니다.

7. **크로스 플랫폼 호환성**

   - Linux, macOS, Windows 등에서 실행됩니다.

8. **프라이버시 중심**
   - 명시적으로 구성되지 않는 한 사용 통계를 수집하거나 데이터를 보내지 않습니다.

### AdGuardPrivate의 새 기능

1. **규칙 목록을 사용한 DNS 라우팅**

   - 구성 파일에 정의된 규칙 목록을 사용하여 DNS 라우팅을 사용자 정의합니다.
   - [Loyalsoldier/v2ray-rules-dat](https://github.com/Loyalsoldier/v2ray-rules-dat)와 같은 타사 규칙을 지원합니다.

2. **응용 프로그램별 차단 규칙 목록**

   - 특정 응용 프로그램의 소스를 차단하도록 구성합니다.
   - 유연한 관리를 위한 타사 구성을 지원합니다.

3. **동적 DNS (DDNS)**

   - 다양한 시나리오에 대한 동적 도메인 이름 해석 기능을 제공합니다.

4. **고급 속도 제한**

   - 효율적인 트래픽 관리 및 제어 조치를 구현합니다.

5. **향상된 배포 기능**
   - 로드 밸런싱 지원.
   - 자동 인증서 유지 관리.
   - 최적화된 네트워크 연결.

자세한 문서를 보려면: [AdGuardPrivate 문서](https://adguardprivate.com/docs/)

## 사용 방법

### 바이너리 다운로드

[Releases](https://github.com/AdGuardPrivate/AdGuardPrivate/releases) 페이지에서 바이너리를 직접 다운로드할 수 있습니다. 다운로드 후 다음 단계를 따라 실행하십시오:

```bash
./AdGuardPrivate -c ./AdGuardHome.yaml -w ./data --web-addr 0.0.0.0:34020 --local-frontend --no-check-update --verbose
```

### Docker 이미지 사용

또는 [Docker Hub](https://hub.docker.com/repository/docker/adguardprivate/adguardprivate)에서 사용할 수 있는 Docker 이미지를 사용할 수 있습니다:

```bash
docker run --rm --name AdGuardPrivate -p 34020:80 -v ./data/container/work:/opt/adguardhome/work -v ./data/container/conf:/opt/adguardhome/conf adguardprivate/adguardprivate:latest
```