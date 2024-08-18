# ClamAction

ClamAction integrates with `clamsmtpd` via the `VirusAction` hook to quarantine suspicious emails, alert administrators, notify recipients, and detail detected threats.

## Features

- Quarantine emails. 
- Send detailed admin alerts.
- Notify recipients with summaries.
- Configurable via environment variables.

## Prerequisites

- **Go**: Install from [golang.org](https://golang.org/dl/).

## Installation

1. **Clone the repository**:

    ```bash
    git clone https://github.com/alexzeitgeist/clamaction.git
    ```

2. **Navigate**:

    ```bash
    cd clamaction
    ```

3. **Build**:

    ```bash
    go build -o clamaction
    ```

## Usage

Configure via environment variables:

- `EMAIL`: Path to the infected email file (supplied by clamsmtpd)
- `RECIPIENTS`: Newline-separated list of intended recipients (supplied by clamsmtpd)
- `SENDER`: Email address of the sender (supplied by clamsmtpd)
- `VIRUS`: Virus ID or description (supplied by clamsmtpd)
- `EMAIL_ADMIN`: Admin email for notifications
- `EMAIL_SERVICE`: Service email for sending notifications
- `QUARANTINE_FOLDER`: Quarantine storage directory
- `SMTP_HOST`: SMTP server host (default `localhost`)
- `SMTP_PORT`: SMTP server port (default `25`)
- `DEBUG`: `true` for verbose logging

### Execution

Configure `clamsmtpd.conf` to invoke ClamAction:

```bash
VirusAction /path/to/clamaction
```

Ensure `clamaction` is executable and correctly located. With environment setup and `clamsmtpd` configured, ClamAction processes emails flagged by ClamAV.
