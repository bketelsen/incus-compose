
## Available Tasks

### build

Description: Build the application
Summary: Build the application with ldflags to set the version with a -dev suffix.

Output: 'incus-compose' in project root.

Run this task:
```
task build
```

### direnv

Description: Add direnv hook to your bashrc

Run this task:
```
task direnv
```

### generate

Description: Generate CLI documentation

Run this task:
```
task generate
```

### tools

Description: Install required tools

Run this task:
```
task tools
```

### checks:all

Description: Run all go checks

Run this task:
```
task checks:all
```

### checks:format

Description: Format all Go source

Run this task:
```
task checks:format
```

### checks:staticcheck

Description: Run go staticcheck

Run this task:
```
task checks:staticcheck
```

### checks:test

Description: Run all tests

Run this task:
```
task checks:test
```

### checks:tidy

Description: Run go mod tidy

Run this task:
```
task checks:tidy
```

### checks:vet

Description: Run go vet on sources

Run this task:
```
task checks:vet
```

### docs:installer

Description: Copy installer from root to site/static directory

Run this task:
```
task docs:installer
```

### docs:site

Description: Run hugo dev server

Run this task:
```
task docs:site
```

### release:goreleaser

Description: Install goreleaser on debian derivatives

Run this task:
```
task release:goreleaser
```

### release:publish

Description: Push and tag at 0.0.1

Run this task:
```
task release:publish
```

### release:release-check

Description: Run goreleaser check

Run this task:
```
task release:release-check
```

### release:snapshot

Description: Run goreleaser in snapshot mode

Run this task:
```
task release:snapshot
```

