# Ramorie CLI Windows Installer
param(
    [string]$InstallPath = "$env:USERPROFILE\bin"
)

# Set up error handling
$ErrorActionPreference = "Stop"

# Colors for console output
function Write-Status {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

function Detect-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    if ($arch -eq "AMD64") {
        return "amd64"
    }
    elseif ($arch -eq "ARM64") {
        return "arm64"
    }
    else {
        Write-Error "Unsupported architecture: $arch"
        exit 1
    }
}

function Get-LatestVersion {
    try {
        Write-Status "Fetching latest release information..."
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/terzigolu/josepshbrain-go/releases/latest"
        $version = $response.tag_name
        Write-Status "Latest version: $version"
        return $version
    }
    catch {
        Write-Error "Failed to get latest version: $_"
        exit 1
    }
}

function Install-Ramorie {
    param([string]$Architecture)

    $binaryName = "ramorie-windows-$Architecture.exe"
    $downloadUrl = "https://github.com/terzigolu/josepshbrain-go/releases/latest/download/$binaryName"

    Write-Status "Downloading ramorie from: $downloadUrl"

    # Create install directory if it doesn't exist
    if (!(Test-Path $InstallPath)) {
        Write-Status "Creating directory: $InstallPath"
        New-Item -ItemType Directory -Path $InstallPath -Force | Out-Null
    }

    $destinationPath = Join-Path $InstallPath "ramorie.exe"

    try {
        # Download the binary
        Invoke-WebRequest -Uri $downloadUrl -OutFile $destinationPath
        Write-Status "Downloaded ramorie to: $destinationPath"
    }
    catch {
        Write-Error "Failed to download ramorie: $_"
        exit 1
    }

    # Add to PATH if not already there
    $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($currentPath -notlike "*$InstallPath*") {
        Write-Status "Adding $InstallPath to user PATH..."
        [Environment]::SetEnvironmentVariable("PATH", "$currentPath;$InstallPath", "User")
        Write-Status "PATH updated. You may need to restart your terminal."
    }

    return $destinationPath
}

function Verify-Installation {
    param([string]$BinaryPath)

    if (Test-Path $BinaryPath) {
        Write-Status "✅ ramorie installed successfully!"

        # Try to get version
        try {
            $version = & $BinaryPath --version 2>$null
            if ($version) {
                Write-Status "Version: $version"
            }
        }
        catch {
            Write-Status "Version: unknown"
        }

        Write-Host ""
        Write-Host "🚀 Get started with:"
        Write-Host "   ramorie setup login"
        Write-Host ""
        Write-Host "📚 For help:"
        Write-Host "   ramorie --help"
        Write-Host ""
        Write-Host "💡 Note: You may need to restart your terminal for PATH changes to take effect."
    }
    else {
        Write-Error "Installation failed. Binary not found at: $BinaryPath"
        exit 1
    }
}

# Main execution
function Main {
    Write-Host "🧠 Ramorie CLI Windows Installer" -ForegroundColor Cyan
    Write-Host "================================" -ForegroundColor Cyan
    Write-Host ""

    $architecture = Detect-Architecture
    Write-Status "Detected architecture: $architecture"

    $version = Get-LatestVersion
    $binaryPath = Install-Ramorie -Architecture $architecture
    Verify-Installation -BinaryPath $binaryPath
}

# Run the installer
try {
    Main
}
catch {
    Write-Error "Installation failed: $_"
    exit 1
}