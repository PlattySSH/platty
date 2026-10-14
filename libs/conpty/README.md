# conpty

Start processes inside a Windows pseudo console (ConPTY), with resize and clean shutdown. On other platforms `Start` returns `ErrUnsupported`.

It runs the local shells in [Platty](https://github.com/PlattySSH/platty).

```sh
go get github.com/plattyssh/conpty
```

```go
pty, err := conpty.Start(conpty.Options{
	Command: []string{"pwsh.exe", "-NoLogo"},
	Cols:    120,
	Rows:    30,
})

if err != nil {
	log.Fatal(err)
}

defer pty.Close()
go io.Copy(os.Stdout, pty)
pty.Resize(100, 40)
code, _ := pty.Wait()
```

Requires Windows 10 1809 or later.

## License

MIT
