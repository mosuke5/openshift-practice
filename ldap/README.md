# LDAPサーバの構築手順

## 概要
このドキュメントでは、PodmanおよびKubernetesを使用してLDAPサーバを構築し、検証を行うための手順を説明します。

テストデータには企業の組織構造（営業本部・システム本部、各部署）を模した実践的なサンプルを使用します。

## 目次
- [Part 1: Podmanを使用したLDAP構築](#part-1-podmanを使用したldap構築)
- [Part 2: Kubernetesを使用したLDAP構築](#part-2-kubernetesを使用したldap構築)
- [OUベースの権限管理について](#ouベースの権限管理について)

---

# Part 1: Podmanを使用したLDAP構築

## 前提条件
- Podmanがインストールされていること

## 1. OpenLDAPコンテナの起動

### 1.1 コンテナの実行
```bash
podman run -d \
  --name openldap \
  -p 1389:389 \
  -p 1636:636 \
  -e LDAP_ORGANISATION="Example Company" \
  -e LDAP_DOMAIN="example.com" \
  -e LDAP_ADMIN_PASSWORD="admin" \
  docker.io/osixia/openldap:latest
```

### パラメータの説明
- `-d`: バックグラウンドで実行
- `--name openldap`: コンテナ名を指定
- `-p 1389:389`: LDAP標準ポート（非暗号化）をホストの1389にマッピング
- `-p 1636:636`: LDAPS標準ポート（SSL/TLS暗号化）をホストの1636にマッピング
- `-e LDAP_ORGANISATION`: 組織名
- `-e LDAP_DOMAIN`: ドメイン名（dc=example,dc=comに変換される）
- `-e LDAP_ADMIN_PASSWORD`: 管理者パスワード

**注意**: ポート番号を1389/1636にすることで、一般ユーザー権限でも使用できます（1024未満のポートは管理者権限が必要）。

## 2. LDAPクライアントツールのインストール

```bash
sudo dnf install openldap-clients
```

## 3. LDAP接続の確認

### 3.1 基本的な接続テスト
```bash
ldapsearch -x -H ldap://localhost:1389 -b "dc=example,dc=com" -D "cn=admin,dc=example,dc=com" -w admin
```

### パラメータの説明
- `-x`: シンプル認証を使用
- `-H`: LDAPサーバのURI
- `-b`: 検索のベースDN（Distinguished Name）
- `-D`: バインドDN（認証に使用するユーザー）
- `-w`: パスワード

## 4. テストデータの追加

### 4.1 テストデータの構造

このリポジトリの `test-users.ldif` には以下の組織構造が含まれています：

```
ou=people
├── ou=sales-division (営業本部)
│   ├── ou=corporate-sales (法人営業部) - 1名
│   └── ou=strategy (戦略部) - 1名
└── ou=it-division (システム本部)
    ├── ou=development (開発部) - 2名
    └── ou=operations (運用部) - 2名
```

**ユーザー一覧（計6名）:**
- 山田太郎（yamada.taro）- 営業本部 法人営業部 課長
- 鈴木花子（suzuki.hanako）- 営業本部 戦略部 戦略企画担当
- 田中次郎（tanaka.jiro）- システム本部 開発部 シニアエンジニア
- 小林真希（kobayashi.maki）- システム本部 開発部 ソフトウェアエンジニア
- 高橋健二（takahashi.kenji）- システム本部 運用部 運用マネージャー
- 渡辺愛（watanabe.ai）- システム本部 運用部 運用エンジニア

### 4.2 データの投入
```bash
ldapadd -x -H ldap://localhost:1389 -D "cn=admin,dc=example,dc=com" -w admin -f test-users.ldif
```

## 5. よく使うLDAPコマンド

### 5.1 全エントリの検索
```bash
ldapsearch -x -H ldap://localhost:1389 -b "dc=example,dc=com" -D "cn=admin,dc=example,dc=com" -w admin "(objectClass=*)"
```

### 5.2 特定のユーザーを検索
```bash
ldapsearch -x -H ldap://localhost:1389 -b "dc=example,dc=com" -D "cn=admin,dc=example,dc=com" -w admin "(uid=yamada.taro)"
```

### 5.3 特定部署のユーザー一覧表示
```bash
# システム本部の全メンバー
ldapsearch -x -H ldap://localhost:1389 -b "ou=it-division,ou=people,dc=example,dc=com" -D "cn=admin,dc=example,dc=com" -w admin "(objectClass=inetOrgPerson)"

# 開発部のメンバーのみ
ldapsearch -x -H ldap://localhost:1389 -b "ou=development,ou=it-division,ou=people,dc=example,dc=com" -D "cn=admin,dc=example,dc=com" -w admin "(objectClass=inetOrgPerson)"
```

### 5.4 エントリの削除
```bash
ldapdelete -x -H ldap://localhost:1389 -D "cn=admin,dc=example,dc=com" -w admin "uid=yamada.taro,ou=corporate-sales,ou=sales-division,ou=people,dc=example,dc=com"
```

### 5.5 エントリの修正
修正用LDIFファイル（`modify.ldif`）を作成：
```ldif
dn: uid=yamada.taro,ou=corporate-sales,ou=sales-division,ou=people,dc=example,dc=com
changetype: modify
replace: title
title: Senior Corporate Sales Manager
```

実行：
```bash
ldapmodify -x -H ldap://localhost:1389 -D "cn=admin,dc=example,dc=com" -w admin -f modify.ldif
```

## 6. LDAP用語集

- **DN (Distinguished Name)**: LDAPエントリの一意の識別子
- **dc (Domain Component)**: ドメインの構成要素
- **ou (Organizational Unit)**: 組織単位。部署や部門などの階層構造を表現
- **cn (Common Name)**: 一般名
- **uid (User ID)**: ユーザーID
- **objectClass**: エントリのタイプを定義
- **LDIF (LDAP Data Interchange Format)**: LDAPデータの標準フォーマット

---

# Part 2: Kubernetesを使用したLDAP構築

## 前提条件
- Kubernetesクラスタが利用可能であること（minikube、kind、OpenShift、本番クラスタなど）
- kubectlコマンドがインストールされていること

## 1. マニフェストファイルの概要

以下のKubernetesリソースを使用してLDAPサーバを構築します：

- **Namespace**: ldap専用の名前空間
- **ConfigMap**: LDAP設定（組織名、ドメインなど）
- **Secret**: 機密情報（管理者パスワード）
- **PersistentVolumeClaim**: データ永続化用のストレージ（2つ）
- **Deployment**: OpenLDAPサーバのデプロイ
- **Service**: LDAPサービスの公開（ClusterIP）
- **Pod**: LDAPクライアントPod

## 2. デプロイ手順

### 2.1 All-in-Oneファイルを使用したデプロイ

すべてのリソースが1つのファイルにまとまっている `ldap-all-in-one.yaml` を使用します：

```bash
# デプロイ
kubectl apply -f ldap-all-in-one.yaml

# デプロイ状態の確認
kubectl get all -n ldap
```

### 2.2 デプロイ状態の確認

```bash
# Podの状態確認
kubectl get pods -n ldap

# Serviceの確認
kubectl get svc -n ldap

# PVCの確認
kubectl get pvc -n ldap

# 詳細情報の確認
kubectl describe pod -n ldap -l app=openldap
```

## 3. テストデータの投入

```bash
# test-users.ldifをクライアントPodにコピー
kubectl cp test-users.ldif ldap/ldap-client:/tmp/test-users.ldif

# クライアントPod内でデータ投入
kubectl exec -n ldap ldap-client -- ldapadd -x -H ldap://openldap:389 -D "cn=admin,dc=example,dc=com" -w admin -f /tmp/test-users.ldif
```

## 4. LDAPクライアントPodを使用した接続

### 4.1 クライアントPodへの接続

LDAPコマンドを実行するための専用クライアントPodが用意されています：

```bash
# クライアントPodに接続
kubectl exec -it -n ldap ldap-client -- bash
```

### 4.2 クライアントPod内からLDAPサーバへ接続

クライアントPod内で以下のコマンドを実行：

```bash
# 接続テスト
ldapsearch -x -H ldap://openldap:389 -b "dc=example,dc=com" -D "cn=admin,dc=example,dc=com" -w admin

# または環境変数を使用
ldapsearch -x -H ldap://${LDAP_SERVER}:389 -b "${LDAP_BASE_DN}" -D "${LDAP_ADMIN_DN}" -w "${LDAP_ADMIN_PASSWORD}"

# システム本部のメンバーを検索
ldapsearch -x -H ldap://openldap:389 -b "ou=it-division,ou=people,dc=example,dc=com" -D "cn=admin,dc=example,dc=com" -w admin "(objectClass=inetOrgPerson)"
```

### 4.3 クライアントPodから直接コマンド実行

Podに入らずに直接コマンドを実行することも可能：

```bash
# 全エントリの検索
kubectl exec -n ldap ldap-client -- ldapsearch -x -H ldap://openldap:389 -b "dc=example,dc=com" -D "cn=admin,dc=example,dc=com" -w admin

# 特定のユーザーを検索
kubectl exec -n ldap ldap-client -- ldapsearch -x -H ldap://openldap:389 -b "dc=example,dc=com" -D "cn=admin,dc=example,dc=com" -w admin "(uid=tanaka.jiro)"
```

### 4.4 クラスタ内の別のPodから接続する場合

他のアプリケーションPodから接続する際の情報：

```yaml
# 接続先ホスト名
openldap.ldap.svc.cluster.local

# ポート
389 (LDAP)
636 (LDAPS)

# 接続例
ldap://openldap.ldap.svc.cluster.local:389
```

---

# OUベースの権限管理について

## 概要

このLDAP構成では、**グループ（groupOfNames）を使用せず、OU（Organizational Unit）の階層構造のみ**を使用してユーザーを管理します。

多くのアプリケーション（AWX、Grafana、Jenkins、GitLabなど）は、OUベースでの権限付与をサポートしています。

## OUベースの権限付与の例

### AWXでの設定例

```python
# LDAPユーザー検索の設定
AUTH_LDAP_USER_SEARCH = LDAPSearch(
    "ou=people,dc=example,dc=com",
    ldap.SCOPE_SUBTREE,
    "(uid=%(user)s)"
)

# システム本部（IT Division）のメンバーに自動的にAdmin権限を付与
AUTH_LDAP_USER_FLAGS_BY_GROUP = {
    "is_superuser": [
        "ou=it-division,ou=people,dc=example,dc=com"
    ]
}

# 開発部のメンバーに特定の権限を付与
AUTH_LDAP_REQUIRE_GROUP = "ou=development,ou=it-division,ou=people,dc=example,dc=com"
```

### 検索フィルタの例

特定のOUに所属する全ユーザーを取得：

```bash
# システム本部の全メンバーを検索
ldapsearch -x -H ldap://localhost:1389 \
  -b "ou=it-division,ou=people,dc=example,dc=com" \
  -D "cn=admin,dc=example,dc=com" -w admin \
  "(objectClass=inetOrgPerson)"

# 開発部のメンバーのみを検索
ldapsearch -x -H ldap://localhost:1389 \
  -b "ou=development,ou=it-division,ou=people,dc=example,dc=com" \
  -D "cn=admin,dc=example,dc=com" -w admin \
  "(objectClass=inetOrgPerson)"
```

## グループ vs OU：設計上の選択

| 観点 | OUベース | グループベース |
|------|----------|----------------|
| **構造** | 組織階層を反映 | 論理的なグルーピング |
| **柔軟性** | 低い（階層構造に固定） | 高い（複数グループに所属可能） |
| **管理** | 組織変更時に影響大 | グループメンバー変更のみ |
| **用途** | 部署単位の権限付与 | 役割単位の権限付与 |
| **例** | 「開発部全員にアクセス権」 | 「管理者グループにAdmin権限」 |

## このリポジトリの設計方針

- **シンプルさ重視**: グループを使わずOUのみで管理
- **組織構造に沿った権限管理**: 部署・部門単位での権限付与を想定
- **実践的な例**: 企業の典型的な組織構造（本部→部署）を模したサンプルデータ

## 実際の権限付与の例

### 例1: システム本部全体に管理者権限

```python
# システム本部（ou=it-division）配下の全ユーザーに管理者権限
base_dn = "ou=it-division,ou=people,dc=example,dc=com"
```

この場合、以下のユーザーが管理者権限を持ちます：
- 田中次郎（開発部）
- 小林真希（開発部）
- 高橋健二（運用部）
- 渡辺愛（運用部）

### 例2: 開発部のみに特定の権限

```python
# 開発部（ou=development）のメンバーにのみデプロイ権限
base_dn = "ou=development,ou=it-division,ou=people,dc=example,dc=com"
```

この場合、以下のユーザーのみが権限を持ちます：
- 田中次郎
- 小林真希

## まとめ

- OUの階層構造を使用することで、組織図に沿った直感的な権限管理が可能
- グループを使用しないことで、LDAP構造がシンプルになり管理が容易
- AWXなどの多くのアプリケーションがOUベースの権限付与をサポート
