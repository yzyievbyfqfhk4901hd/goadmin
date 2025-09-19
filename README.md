## Conclusion
- Use C# for this shit.
- Dont use GO for this shit.
- Misusing adm toolkits will make u 0$ and **will** get u in jail.
- Use responsibly and only on machines you own or have explicit permission to access.

## What it does

This bot gives you the power to:
- Take screenshots of all your monitors (or just the main one, whatever)
- Record video of your screen (5 seconds, because longer would be too much)
- Record audio from your microphone (10 seconds, same reasoning)
- Kill processes that are misbehaving
- Send popup messages to annoy yourself
- Get system info and hardware details
- Kill browser processes when watching porn

## Requirements

- Go
- FFmpeg
- A Telegram bot token (get one from @BotFather)
- A computer

## Setup

1. Create `secrets.json` for ur tokens and shit:
```json
{
  "authorized_users": [
    123456789
  ],
  "bot_token": "token"
}
```

2. Create `banned.json` for ur banned sites and shit:
(only for tab names, not literall urls.)
```json
{
  "banned_sites": [
    "pornhub",
    "",
    ""
  ]
}
```

3. Get a telegram bot token
4. Get telegram ID ready
5. Run `go mod tidy` to get dependencies
6. Run `go run main.go`

## Commands

- `/start` - Shows something
- `/help` - I wonder what it does
- `/info` - System information and hardware details
- `/ss` - Take screenshots of all monitors
- `/ssa` - Take a screenshot of all monitors as one image
- `/ssm` - Take a screenshot of just the main monitor
- `/vid` - Record 5 seconds of video (requires FFmpeg)
- `/audio` - Record 10 seconds of audio (requires FFmpeg)
- `/processes` - List running processes
- `/kill <PID>` - Kill a process by name
- `/browser` - Browser monitoring commands (start/stop/status/list)
- `/msg <message>` - Send a popup message to the computer
- `/displays` - Show display information
- `/files` - Show supported file types