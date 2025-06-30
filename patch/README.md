# Apply patch to encore

```bash
git apply --stat patch/enable-plugin-directives.patch
git apply --check patch/enable-plugin-directives.patch
```

## Apply patch to encore

```bash
git apply patch/enable-plugin-directives.patch

git diff           # review changes
git add v2/parser/apis/directive/directive.go
git commit -m "Make //encore:… directives pluggable via RegisterDirectiveParser"
```

## Rebuild

```bash
go build -o ~/.local/bin/encore ./cli/cmd/encore
```