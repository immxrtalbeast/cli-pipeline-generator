package generator

import (
    "fmt"
    "strings"

    "github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generatePHPPipeline(info *analyzer.ProjectInfo) string {
    var pipeline strings.Builder

    buildTool := "composer"
    if info.BuildTool != "composer" && info.BuildTool != "php" {
        buildTool = info.BuildTool
    }

    pipeline.WriteString(fmt.Sprintf(`name: PHP CI/CD Pipeline (%s)

on:
  push:
    branches: [ main, master, develop ]
  pull_request:
    branches: [ main, master, develop ]

env:
  PHP_VERSION: '`, buildTool))

    // Устанавливаем версию PHP
    if info.Version != "" {
        pipeline.WriteString(info.Version)
    } else {
        pipeline.WriteString("8.1")
    }

    pipeline.WriteString(`'

jobs:
`)

    // Job для установки зависимостей
    pipeline.WriteString(`  setup:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Setup PHP
      uses: shivammathur/setup-php@v2
      with:
        php-version: ${{ env.PHP_VERSION }}
        extensions: mbstring, xml, ctype, iconv, intl, pdo_sqlite
        coverage: none
        
    - name: Check PHP version
      run: php -v
`)

    // Если есть Composer, устанавливаем зависимости
    if info.PackageManager == "composer" {
        pipeline.WriteString(`    - name: Validate composer.json and composer.lock
      run: composer validate --strict
      
    - name: Cache Composer packages
      id: composer-cache
      uses: actions/cache@v3
      with:
        path: vendor
        key: ${{ runner.os }}-php-${{ hashFiles('**/composer.lock') }}
        restore-keys: |
          ${{ runner.os }}-php-
          
    - name: Install dependencies
      run: composer install --prefer-dist --no-progress --no-interaction
`)
    }

    // Job для линтинга и статического анализа
    pipeline.WriteString(`  lint:
    runs-on: ubuntu-latest
    needs: setup
    steps:
    - uses: actions/checkout@v3
    - name: Setup PHP
      uses: shivammathur/setup-php@v2
      with:
        php-version: ${{ env.PHP_VERSION }}
        
`)

    if info.PackageManager == "composer" {
        pipeline.WriteString(`    - name: Install dependencies
      run: composer install --prefer-dist --no-progress --no-interaction
      
`)
    }

    // Проверка синтаксиса
    pipeline.WriteString(`    - name: Check PHP syntax
      run: find . -name "*.php" -exec php -l {} \;
`)

    // Статический анализ если есть соответствующие зависимости
    if containsDependency(info.Dependencies, "quality:static-analysis") {
        pipeline.WriteString(`    - name: Run PHPStan
      run: composer require --dev phpstan/phpstan && vendor/bin/phpstan analyse
`)
    }

    if containsDependency(info.Dependencies, "quality:code-style") {
        pipeline.WriteString(`    - name: Check code style with PHP_CodeSniffer
      run: composer require --dev squizlabs/php_codesniffer && vendor/bin/phpcs
`)
    }

    // Job для тестов
    if info.HasTests {
        pipeline.WriteString(`  test:
    runs-on: ubuntu-latest
    needs: setup
    strategy:
      matrix:
        php-version: ['8.1', '8.2', '8.3']
    steps:
    - uses: actions/checkout@v3
    
    - name: Setup PHP
      uses: shivammathur/setup-php@v2
      with:
        php-version: ${{ matrix.php-version }}
        extensions: mbstring, xml, ctype, iconv, intl, pdo_sqlite
        coverage: pcov
        
`)

        if info.PackageManager == "composer" {
            pipeline.WriteString(`    - name: Install dependencies
      run: composer install --prefer-dist --no-progress --no-interaction
      
`)
        }

        // Запуск тестов в зависимости от фреймворка
        switch info.TestFramework {
        case "phpunit":
            pipeline.WriteString(`    - name: Run PHPUnit tests
      run: |
        if [ -f vendor/bin/phpunit ]; then
          vendor/bin/phpunit --coverage-clover=coverage.xml
        elif [ -f phpunit ]; then
          ./phpunit --coverage-clover=coverage.xml
        else
          phpunit --coverage-clover=coverage.xml
        fi
        
    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
      with:
        file: coverage.xml
`)

        case "pest":
            pipeline.WriteString(`    - name: Run Pest tests
      run: vendor/bin/pest --coverage-clover=coverage.xml
      
    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
      with:
        file: coverage.xml
`)

        case "codeception":
            pipeline.WriteString(`    - name: Run Codeception tests
      run: vendor/bin/codecept run --coverage-xml
`)

        default:
            pipeline.WriteString(`    - name: Run tests
      run: |
        if [ -f vendor/bin/phpunit ]; then
          vendor/bin/phpunit
        else
          echo "No test runner found"
        fi
`)
        }

        // Для Laravel проектов
        if containsDependency(info.Dependencies, "framework:laravel") {
            pipeline.WriteString(`    - name: Laravel environment setup
      run: |
        cp .env.example .env.test
        php artisan key:generate --env=test
        
    - name: Run Laravel specific tests
      run: php artisan test
`)
        }
    }

    // Job для сборки (если требуется)
    previousJob := "setup"
    if info.HasTests {
        previousJob = "test"
    }

    pipeline.WriteString(fmt.Sprintf(`  build:
    runs-on: ubuntu-latest
    needs: %s
    if: github.event_name == 'push' && (github.ref == 'refs/heads/main' || github.ref == 'refs/heads/master')
    steps:
    - uses: actions/checkout@v3
    - name: Setup PHP
      uses: shivammathur/setup-php@v2
      with:
        php-version: ${{ env.PHP_VERSION }}
        
`, previousJob))

    if info.PackageManager == "composer" {
        pipeline.WriteString(`    - name: Install dependencies (no dev)
      run: composer install --prefer-dist --no-dev --no-progress --no-interaction --optimize-autoloader
      
    - name: Optimize autoloader
      run: composer dump-autoload --no-dev --optimize
`)
    }

    // Для Laravel проектов
    if containsDependency(info.Dependencies, "framework:laravel") {
        pipeline.WriteString(`    - name: Laravel optimization
      run: |
        php artisan config:cache
        php artisan route:cache
        php artisan view:cache
        
    - name: Generate build artifacts
      run: |
        mkdir -p build
        cp -r app bootstrap config database public resources routes storage vendor build/
        cp artisan composer.json composer.lock .env.example build/
`)
    } else {
        pipeline.WriteString(`    - name: Create build package
      run: |
        mkdir -p build
        cp -r src vendor build/
        if [ -f composer.json ]; then cp composer.json composer.lock build/; fi
`)
    }

    pipeline.WriteString(`    - name: Upload build artifacts
      uses: actions/upload-artifact@v3
      with:
        name: php-build
        path: build/
`)

    // Job для деплоя (опционально)
    if containsDependency(info.Dependencies, "framework:laravel") {
        pipeline.WriteString(`  deploy:
    runs-on: ubuntu-latest
    needs: build
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    steps:
    - name: Download build artifacts
      uses: actions/download-artifact@v3
      with:
        name: php-build
        
    - name: Deploy to server
      env:
        DEPLOY_KEY: ${{ secrets.DEPLOY_KEY }}
        SERVER_HOST: ${{ secrets.SERVER_HOST }}
      run: |
        echo "Deployment would happen here"
        # rsync -avz -e "ssh -i $DEPLOY_KEY" build/ user@$SERVER_HOST:/var/www/html/
`)
    }

    // Job для безопасности
    pipeline.WriteString(`  security:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Setup PHP
      uses: shivammathur/setup-php@v2
      with:
        php-version: ${{ env.PHP_VERSION }}
        
`)

    if info.PackageManager == "composer" {
        pipeline.WriteString(`    - name: Install dependencies
      run: composer install --prefer-dist --no-progress --no-interaction
      
    - name: Security check with Symfony Security Checker
      run: composer require --dev enlightn/security-checker && vendor/bin/security-checker security:check
      
    - name: Check for vulnerable dependencies
      run: composer audit
`)
    }

    return pipeline.String()
}