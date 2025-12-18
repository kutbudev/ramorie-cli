# JosephsBrain CLI – User Registration & Quick Start Guide

## 1. Installation

Build and install the CLI globally:

```sh
go build -o jbraincli cmd/jbraincli/main.go
sudo cp jbraincli /usr/local/bin/
```

Check installation:

```sh
jbraincli --help
```

---

## 2. First-Time Setup

### a. Login / Register

Authenticate your CLI with your API key:

```sh
jbraincli setup login
```

- Enter your API key when prompted.
- The CLI stores credentials in `~/.jbrain/config.json`.

**Manual config:**
Alternatively, create the config file directly:

```sh
mkdir -p ~/.jbrain
echo '{"api_key": "YOUR_API_KEY"}' > ~/.jbrain/config.json
```

---

## 3. Usage Examples

### List Projects

```sh
jbraincli project list
```

### Create a Project

```sh
jbraincli project create "My Project"
```

### Create a Task

```sh
jbraincli task create --project "My Project" "Task description" "Task title"
```

### Create a Memory

```sh
jbraincli memory create --project "My Project" "Memory content"
```

---

## 4. Troubleshooting

- If you see `Permission denied`, ensure the binary is executable:
  `chmod +x /usr/local/bin/jbraincli`
- For API/auth errors, check your API key in `~/.jbrain/config.json`.

---

## 5. More Help

For all commands:

```sh
jbraincli --help
```

Or see the README.md for advanced usage.
