$base = "C:\Users\fbolz\Documents\Code\ProRouter"
$dirs = @(
    "gateway-go\cmd\prorouter",
    "gateway-go\internal\adapters",
    "gateway-go\internal\cache",
    "gateway-go\internal\config",
    "gateway-go\internal\database",
    "gateway-go\internal\middleware",
    "gateway-go\internal\migrations",
    "gateway-go\internal\models",
    "gateway-go\internal\oauth",
    "gateway-go\internal\proxy",
    "gateway-go\internal\routing",
    "gateway-go\internal\tokenizer",
    "dashboard-zen\src\app\dashboard",
    "dashboard-zen\src\app\keys",
    "dashboard-zen\src\app\providers",
    "dashboard-zen\src\app\recipes",
    "dashboard-zen\src\app\playground",
    "dashboard-zen\src\app\settings",
    "dashboard-zen\src\components\ui",
    "dashboard-zen\src\components\prorouter",
    "dashboard-zen\src\lib",
    "dashboard-zen\public",
    "cli-npm\bin",
    "cli-npm\scripts",
    "cli-npm\src",
    "scripts",
    ".github\workflows",
    ".github\ISSUE_TEMPLATE"
)
foreach ($d in $dirs) {
    $path = Join-Path $base $d
    New-Item -ItemType Directory -Path $path -Force | Out-Null
}
Write-Host "All directories created."
