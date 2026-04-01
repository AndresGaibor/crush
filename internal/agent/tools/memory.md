# Memory Tool

Save, load, search, and manage persistent memories across sessions.

Memories help Crush remember important information between conversations, such as:
- Project conventions and coding style preferences
- Architecture decisions and patterns
- Frequently used commands and configurations
- User preferences that have been corrected multiple times

## Usage

### Save a memory
```
action: save
id: project-conventions
content: |
  ## Project Conventions
  - Use tabs for indentation
  - Follow REST naming conventions
  - All responses in Spanish
scope: project
tags: [conventions, style]
```

### List all memories
```
action: list
```

### Search memories
```
action: search
query: project conventions coding style
```

### Load a specific memory
```
action: load
id: project-conventions
```

### Get memory statistics
```
action: stats
```

### Check for auto-suggested memories
```
action: suggest
```
