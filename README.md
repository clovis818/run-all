# run-all

`run-all` is a command-line tool that allows you to execute commands across multiple directories based on a customizable directory pattern. It is designed for developers and sysadmins who need to run repetitive tasks in multiple project folders, offering options to exclude directories, run commands in parallel, and more.

## Features

- **Directory Pattern Matching**: Run commands across directories that match a specified pattern.
- **Parallel Execution**: Execute commands in multiple directories simultaneously to save time.
- **Conditional Execution**: Require a specific file or folder to be present in a directory for it to be included.
- **Exclude Directories**: Specify directories or patterns to exclude from execution.
- **Dry Run**: Test what will happen without actually running the commands.
- **Error Handling**: Optionally continue running commands in other directories even if one command fails.

## Installation

Clone the repository and build the tool:

```bash
git clone https://github.com/clovis818/run-all.git
cd run-all
go build -o run-all cmd/run-all/main.go
```

Alternatively, you can download the latest release from the [Releases](https://github.com/clovis818/run-all/releases) page.

## Usage
```bash
run-all [options]
```

### Available Options

- `-command string`  
  Command(s) to run in each matched directory.

- `-continue-on-failure`  
  Continue executing commands in other directories even if one fails.

- `-dir-pattern string`  
  Directory pattern to match.

- `-dry-run`  
  Perform a dry run without executing commands.

- `-exclude string`  
  Comma-separated list of patterns of directories to exclude (relative to dir-pattern or full path).

- `-parallel`  
  Run commands in parallel.

- `-require string`  
  Required folder or file for a directory to be included.

## Examples

### 1. Git Status in All Go Modules

Run `git status` in all directories under `/projects/api/` that contain a `go.mod` file:

```bash
run-all -dir-pattern="/projects/api/*" -require="go.mod" -parallel -command="git status"
```

### 2. Update Poetry in All Python Packages

Run `poetry update` in all directories under `/projects/api/*/test`:

```bash
run-all -dir-pattern="/projects/api/*/test" -parallel -command="poetry update"
```

### 3. Clean Up Logs in All Log Directories

Remove all `.log` files in directories matching `/var/logs/*`, excluding directories named `backup`:

```bash
run-all -dir-pattern="/var/logs/*" -exclude="backup" -command="rm *.log"
```

### 4. Perform a Dry Run for Deployment

See what would happen if you deploy to all project directories, without actually executing the commands:

```bash
run-all -dir-pattern="/apps/*" -require="Dockerfile" -dry-run -command="docker-compose up -d"
```

## Contributing

Feel free to fork the repository, create a branch, and submit a pull request. Contributions are welcome!

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
