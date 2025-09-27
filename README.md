# cli-pipeline-generator

[![wakatime](https://wakatime.com/badge/user/42cf6868-b638-4d34-9e52-ec8f63476139/project/a0dc55d5-6a53-4c68-8d8b-ee72ef20ca24.svg)](https://wakatime.com/badge/user/42cf6868-b638-4d34-9e52-ec8f63476139/project/a0dc55d5-6a53-4c68-8d8b-ee72ef20ca24)
# Описание

Данная утилита позволяет генерировать pipeline для ci/cd на основе предаставленного репозитория(есть поддержка удаленного репозитория с github). Реализована поддержка 10+ языков(Go, Python, Java, PHP, Rust...) Работает с форматами gitlab, github actions и jenkins

# Использование

Удаленный репозиторий
```
pipeline-gen --remote {ссылка на репу} --branch {ветка репозитория} --output {название файла для пайплана.yml}
```

Локальный репозиторий
```
pipeline-gen --repo {путь до репозитория} --output {название файла для пайплана.yml}
```
Флаги
```
Flags:
  -b, --branch string   Branch to analyze (default "main")
  -f, --format string   CI/CD format (github, gitlab, jenkins) (default "github")
  -h, --help            help for pipeline-gen
  -o, --output string   Output pipeline file (default "pipeline.yml")
  -R, --remote string   URL of remote git repository
  -r, --repo string     Path to local repository
```
