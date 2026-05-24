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
│   ├── demo_failure.robot   # échec volontaire (alerte QA Capsule en CI)
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

## Lancer depuis un pipeline CI/CD

Le point d’entrée unique est **`scripts/run-tests.sh`** : il installe les deps, exécute Robot, convertit en JUnit, puis appelle QA Capsule si les secrets sont présents.

### Prérequis côté QA Capsule

1. Instance QA Capsule **accessible depuis Internet** (ou réseau du runner) — `localhost` ne fonctionne que sur un runner self-hosted.
2. Dans **CI/CD Gateways** : copier la **clé API** du projet.
3. URL d’upload : `{QA_CAPSULE_URL}/api/webhooks/upload?framework=RobotFramework`

### Variables d’environnement du pipeline

| Variable | Obligatoire | Exemple |
|----------|-------------|---------|
| `QA_CAPSULE_URL` | Oui (pour upload) | `https://qa-capsule.example.com` |
| `QA_CAPSULE_API_KEY` | Oui (pour upload) | `sk-...` |
| `CI_PIPELINE_ID` | Recommandé | ID du job (→ `X-Run-Id`) |
| `QA_CAPSULE_EXEC_ENV` | Non | `STAGING`, `PROD`, `DEV` |
| `QA_CAPSULE_EXEC_TYPE` | Non | `TEST-RUN`, `SMOKE`, `NIGHTLY` |
| `SELENIUM_ENABLED` | Non | `true` seulement si Chrome/WebDriver sur le runner |

Sans `QA_CAPSULE_*`, les tests tournent quand même ; l’upload est ignoré (utile pour valider le job avant de brancher les secrets).

---

### GitHub Actions

Workflow aligné sur **API Pipeline (Pytest)** et **E2E Playwright** : [`.github/workflows/robot-tests.yml`](../.github/workflows/robot-tests.yml)

1. **Settings → Secrets and variables → Actions** → ajouter :
   - `QA_CAPSULE_URL` — base URL (sans `/` final), ex. `https://qa-capsule.example.com`
   - `QA_CAPSULE_API_ROBOT_KEY` — clé du projet (ou `QA_CAPSULE_API_KEY`)
2. Lancer : **Actions → Robot Framework Pipeline → Run workflow** (ou push sur `main`).

Même schéma que les autres pipelines du dépôt :

| Secret | Exemple Pytest | Robot |
|--------|----------------|-------|
| URL | `QA_CAPSULE_URL` | `QA_CAPSULE_URL` |
| Clé API | `QA_CAPSULE_API_PYTEST_KEY` | `QA_CAPSULE_API_ROBOT_KEY` |

Extrait minimal si vous avez déjà un job :

```yaml
jobs:
  robot:
    runs-on: ubuntu-latest
    env:
      QA_CAPSULE_URL: ${{ secrets.QA_CAPSULE_URL }}
      QA_CAPSULE_API_KEY: ${{ secrets.QA_CAPSULE_API_KEY }}
      CI_PIPELINE_ID: ${{ github.run_id }}
      QA_CAPSULE_EXEC_ENV: STAGING
      QA_CAPSULE_EXEC_TYPE: TEST-RUN
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: '3.12'
      - run: chmod +x scripts/run-tests.sh && ./scripts/run-tests.sh
```

---

### GitLab CI

Ajoutez un job dans `.gitlab-ci.yml` :

```yaml
robot-tests:
  image: python:3.12
  stage: test
  variables:
    QA_CAPSULE_EXEC_ENV: STAGING
    QA_CAPSULE_EXEC_TYPE: TEST-RUN
    CI_PIPELINE_ID: $CI_PIPELINE_ID
    SELENIUM_ENABLED: "false"
  before_script:
    - apt-get update -qq && apt-get install -y -qq curl
  script:
    - chmod +x scripts/run-tests.sh
    - ./scripts/run-tests.sh
  artifacts:
    when: always
    paths:
      - tests/results/
    expire_in: 7 days
```

Secrets GitLab : **Settings → CI/CD → Variables** (masquées) :

- `QA_CAPSULE_URL`
- `QA_CAPSULE_API_KEY`

---

### Azure DevOps / Jenkins (principe identique)

```bash
export QA_CAPSULE_URL="$(QA_CAPSULE_URL)"
export QA_CAPSULE_API_KEY="$(QA_CAPSULE_API_KEY)"
export CI_PIPELINE_ID="$(Build.BuildId)"   # Azure
# export CI_PIPELINE_ID="$BUILD_NUMBER"    # Jenkins
chmod +x scripts/run-tests.sh
./scripts/run-tests.sh
```

Les résultats apparaissent dans **Operations → Telemetry Stream** (groupe par `pipeline_run_id` = ID du pipeline).
