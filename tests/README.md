# Tests automatisés (Robot Framework)

Suite d’exemples exécutables en local ou en CI/CD, avec envoi des résultats vers **QA Capsule** au format JUnit XML.

## Prérequis

- Python 3.10+
- `bash` (Git Bash sur Windows, ou Linux/macOS)
- Pour `ui_navigation.robot` : Chrome/Chromium + WebDriver, et `SELENIUM_ENABLED=true`

## Structure

```
tests/
├── requirements.txt
├── robotframework/
│   ├── resources/common.robot
│   ├── smoke_tests.robot
│   ├── api_health.robot
│   └── ui_navigation.robot
└── results/          # généré (ignoré par git)
```

## Lancer en local

Depuis la racine du dépôt :

```bash
chmod +x scripts/run-tests.sh
./scripts/run-tests.sh
```

Sans variables QA Capsule, seuls les tests Robot sont exécutés ; l’upload est ignoré.

### Upload vers QA Capsule

```bash
export QA_CAPSULE_URL="http://localhost:9000"
export QA_CAPSULE_API_KEY="votre-clé-api-projet"
export QA_CAPSULE_EXEC_ENV="DEV"
export QA_CAPSULE_EXEC_TYPE="TEST-RUN"

./scripts/run-tests.sh
```

### Cible API (optionnel)

Par défaut `api_health.robot` utilise `jsonplaceholder.typicode.com`. Pour votre API :

```bash
export API_HEALTH_HOST=api.example.com
./scripts/run-tests.sh
```

### Tests UI (optionnel)

```bash
export SELENIUM_ENABLED=true
export SELENIUM_BROWSER=headlesschrome
./scripts/run-tests.sh
```

## CI/CD (exemple)

Injectez `QA_CAPSULE_URL` et `QA_CAPSULE_API_KEY` comme secrets du pipeline, puis :

```yaml
- run: chmod +x scripts/run-tests.sh && ./scripts/run-tests.sh
  env:
    QA_CAPSULE_URL: ${{ secrets.QA_CAPSULE_URL }}
    QA_CAPSULE_API_KEY: ${{ secrets.QA_CAPSULE_API_KEY }}
    CI_PIPELINE_ID: ${{ github.run_id }}
```

Les rapports apparaissent dans **Operations → Telemetry Stream** (matrice d’exécution + incidents sur les échecs).
