package generator

import (
  "fmt"
  "strings"

  "github.com/immxrtalbeast/pipeline-gen/internal/analyzer"
)

func generateCSharpPipeline(info *analyzer.ProjectInfo) string {
  var pipeline strings.Builder

  pipeline.WriteString(`name: .NET CI/CD Pipeline

on:
  push:
    branches: [ main, master, develop ]
  pull_request:
    branches: [ main, master, develop ]

jobs:
`)

  pipeline.WriteString(`  setup:
    runs-on: windows-latest
    steps:
    - uses: actions/checkout@v3
    - name: Setup .NET
      uses: actions/setup-dotnet@v4
      with:
        dotnet-version: '`)
  if info.Version != "" {
    pipeline.WriteString(info.Version)
  } else {
    pipeline.WriteString("8.0.x")
  }
  pipeline.WriteString(`'
`)

  pipeline.WriteString(`  restore:
    runs-on: windows-latest
    needs: setup
    steps:
    - uses: actions/checkout@v3
    - name: Setup .NET
      uses: actions/setup-dotnet@v4
      with:
        dotnet-version: '`)
  if info.Version != "" {
    pipeline.WriteString(info.Version)
  } else {
    pipeline.WriteString("8.0.x")
  }
  pipeline.WriteString(`'
    - name: Restore dependencies
      run: dotnet restore
`)

  if info.HasTests {
    pipeline.WriteString(`  test:
    runs-on: windows-latest
    needs: restore
    strategy:
      matrix:
        dotnet: [`)
    if info.Version != "" && info.Version != "8.0" {
      pipeline.WriteString(fmt.Sprintf(" '%s.x', '8.0.x' ", info.Version))
    } else {
      pipeline.WriteString(" '7.0.x', '8.0.x' ")
    }
    pipeline.WriteString(`]
    steps:
    - uses: actions/checkout@v3
    - name: Setup .NET ${{ matrix.dotnet }}
      uses: actions/setup-dotnet@v4
      with:
        dotnet-version: ${{ matrix.dotnet }}
    - name: Restore
      run: dotnet restore
    - name: Build
      run: dotnet build --no-restore --configuration Release
    - name: Test
      run: `)
    switch info.TestFramework {
    case "mstest":
      pipeline.WriteString("dotnet test --no-build --configuration Release --logger trx")
    case "nunit":
      pipeline.WriteString("dotnet test --no-build --configuration Release --logger trx")
    case "xunit":
      pipeline.WriteString("dotnet test --no-build --configuration Release --logger trx")
    default:
      pipeline.WriteString("dotnet test --no-build --configuration Release")
    }
    pipeline.WriteString(`
`)
  } else {
    pipeline.WriteString(`  verify:
    runs-on: windows-latest
    needs: restore
    steps:
    - uses: actions/checkout@v3
    - name: Setup .NET
      uses: actions/setup-dotnet@v4
      with:
        dotnet-version: '`)
    if info.Version != "" {
      pipeline.WriteString(info.Version)
    } else {
      pipeline.WriteString("8.0.x")
    }
    pipeline.WriteString(`'
    - name: Restore
      run: dotnet restore
    - name: Build
      run: dotnet build --no-restore --configuration Release
`)
  }

  prev := "test"
  if !info.HasTests {
    prev = "verify"
  }

  pipeline.WriteString(fmt.Sprintf(`  publish:
    runs-on: windows-latest
    needs: %s
    steps:
    - uses: actions/checkout@v3
    - name: Setup .NET
      uses: actions/setup-dotnet@v4
      with:
        dotnet-version: '`, prev))
  if info.Version != "" {
    pipeline.WriteString(info.Version)
  } else {
    pipeline.WriteString("8.0.x")
  }
  pipeline.WriteString(`'
    - name: Restore
      run: dotnet restore
    - name: Build
      run: dotnet build --no-restore --configuration Release
`)

  // Publish for ASP.NET Core if detected
  if containsDependency(info.Dependencies, "web-framework:aspnetcore") {
    pipeline.WriteString(`    - name: Publish
      run: dotnet publish --no-build --configuration Release -o out
    - name: Upload artifacts
      uses: actions/upload-artifact@v3
      with:
        name: publish
        path: out/
`)
  } else {
    pipeline.WriteString(`    - name: Pack (NuGet)
      run: dotnet pack --no-build --configuration Release -o packages
    - name: Upload artifacts
      uses: actions/upload-artifact@v3
      with:
        name: packages
        path: packages/
`)
  }

  return pipeline.String()
}
