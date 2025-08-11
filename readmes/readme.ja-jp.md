# NullPrivate

NullPrivateは、_AdGuardHome_のフォークで、強化された機能とカスタマイズ性を備えたSaaSホストバージョンを提供するために設計されています。ホスティングは[Null Private](https://nullprivate.com)で行われています。

## 主要な機能

### オリジナルの機能

1. **ネットワーク全体の広告ブロック**

   - ネットワーク内のすべてのデバイスで広告とトラッカーをブロックします。
   - トラッキングドメインを「ブラックホール」に再ルーティングするDNSサーバーとして動作します。

2. **カスタムフィルタリングルール**

   - 独自のカスタムフィルタリングルールを追加します。
   - ネットワークアクティビティを監視および制御します。

3. **暗号化されたDNSサポート**

   - DNS-over-HTTPS、DNS-over-TLS、DNSCryptをサポートします。

4. **組み込みのDHCPサーバー**

   - すぐに使えるDHCPサーバー機能を提供します。

5. **クライアントごとの設定**

   - 個々のデバイスに対する設定を構成します。

6. **ペアレンタルコントロール**

   - アダルトドメインをブロックし、検索エンジンでセーフサーチを強制します。

7. **クロスプラットフォームの互換性**

   - Linux、macOS、Windowsなどで動作します。

8. **プライバシーに焦点を当てた**
   - 明示的に設定されていない限り、使用統計を収集したりデータを送信したりしません。

### NullPrivateによる新機能

1. **ルールリストを使用したDNSルーティング**

   - 設定ファイルで定義されたルールリストを使用してDNSルーティングをカスタマイズします。
   - [Loyalsoldier/v2ray-rules-dat](https://github.com/Loyalsoldier/v2ray-rules-dat)などのサードパーティのルールをサポートします。

2. **アプリケーション固有のブロックルールリスト**

   - 特定のアプリケーションからのソースのブロックを設定します。
   - 柔軟な管理のためのサードパーティの設定をサポートします。

3. **ダイナミックDNS (DDNS)**

   - さまざまなシナリオでのダイナミックドメイン名解決機能を提供します。

4. **高度なレート制限**

   - 効率的なトラフィック管理と制御措置を実施します。

5. **強化されたデプロイ機能**
   - ロードバランシングのサポート。
   - 自動証明書のメンテナンス。
   - 最適化されたネットワーク接続。

詳細なドキュメントはこちらを訪問してください：[NullPrivate ドキュメント](https://nullprivate.com/docs/)

## 使用方法

### バイナリのダウンロード

バイナリは[Releases](https://github.com/NullPrivate/NullPrivate/releases)ページから直接ダウンロードできます。ダウンロード後、以下の手順で実行します：

```bash
./NullPrivate -c ./AdGuardHome.yaml -w ./data --web-addr 0.0.0.0:34020 --local-frontend --no-check-update --verbose
```

### Dockerイメージの使用

または、[Docker Hub](https://hub.docker.com/repository/docker/nullprivate/nullprivate)で利用可能なDockerイメージを使用できます：

```bash
docker run --rm --name NullPrivate -p 34020:80 -v ./data/container/work:/opt/adguardhome/work -v ./data/container/conf:/opt/adguardhome/conf nullprivate/nullprivate:latest
```