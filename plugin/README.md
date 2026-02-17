Here’s a sketch of how you could patch **`v2/parser/apis/directive/directive.go`** itself to let you register custom directive‐parsers from *your* code, without having to touch the core list of built‐ins every time. The idea is:

1. Add a global slice of plugin callbacks and a `RegisterDirectiveParser` function.
2. In `Parse`, after you’ve parsed the built‐in directive, invoke any plugin parsers that match the same name.
3. In your own module, you write a small package whose `init()` does a blank‐import into `cmd/encore/main.go` and calls `parser.RegisterDirectiveParser("pubsub", myPubSubParser)`.

---

### 1) Patch `directive.go`

```diff
--- a/v2/parser/apis/directive/directive.go
+++ b/v2/parser/apis/directive/directive.go
@@   // Directive represents a parsed "encore:" directive.
 type Directive struct { /* … */ }

+// DirectiveParser is a callback signature for handling a Directive on a FuncDecl.
+type DirectiveParser func(d *Directive, decl *ast.FuncDecl) error
+
+// pluginParsers holds user‐registered parsers.
+var pluginParsers = map[string]DirectiveParser{}

 func init() {
     // nothing to change here for built‐ins…
 }
@@
 // Parse parses the encore:foo directives in cg.
 // It returns the parsed directives, if any, and the
 // remaining doc text after stripping the directive lines.
 func Parse(errs *perr.List, cg *ast.CommentGroup) (dir *Directive, doc string, ok bool) {
     // … existing code that finds exactly one built‐in directive in cg …
 
     // Suppose at this point we have `dir` and the original FuncDecl `decl`.
-    return dirs[0], doc, true
+    // Now, if the user has registered a plugin handler for this directive name, invoke it.
+    if parser, found := pluginParsers[dir.Name]; found {
+        // you’ll need to obtain the FuncDecl here – in the real parser you have access
+        // to it; let’s assume it’s passed in as a parameter to Parse:
+        if err := parser(dirs[0], decl); err != nil {
+            errs.Add(perr.Wrap(err))
+            return nil, "", false
+        }
+    }
+    return dirs[0], doc, true
 }
```

And **above** or **below** that in the same file (outside of any function), add:

```go
// RegisterDirectiveParser lets your own package hook into `//encore:<name>`
// without editing the core built-ins.
func RegisterDirectiveParser(name string, parser DirectiveParser) {
    if name == "" || parser == nil {
        panic("invalid plugin directive registration")
    }
    pluginParsers[name] = parser
}
```

---

### 2) Write your plugin in your own module

In, say, `github.com/you/mypubsub/directive.go`:

```go
package mypubsub

import (
    "fmt"
    "go/ast"
    "strings"

    "github.com/encoredev/encore/v2/parser/apis/directive"
)

func init() {
    directive.RegisterDirectiveParser("pubsub", parsePubSub)
}

func parsePubSub(d *directive.Directive, decl *ast.FuncDecl) error {
    args := strings.Fields(strings.Join(d.Fields, " "))
    if len(args) != 1 {
        return fmt.Errorf("pubsub directive requires exactly one subject")
    }
    subject := args[0]
    // validate signature: func(context.Context, *T) error …
    // then annotate or stash metadata for codegen. For example:
    decl.Doc.List = append(decl.Doc.List, &ast.Comment{
        Text: "// @encore:pubsub:" + subject,
    })
    return nil
}
```

### 3) Import your plugin into the Encore CLI

In `cmd/encore/main.go`, add a blank import so your plugin’s `init()` runs:

```go
import (
    // … existing imports …
    _ "github.com/you/mypubsub"
)
```

Then rebuild:

```bash
cd path/to/encore
go build -o $ENCORE_INSTALL/bin/encore ./cli/cmd/encore
go build -o $ENCORE_INSTALL/bin/git-remote-encore ./cli/cmd/git-remote-encore
```

Now, whenever the parser sees

```go
//encore:pubsub orders.created
func HandleOrderCreated(ctx context.Context, evt *OrderCreated) error { … }
```

it will first parse it with the core logic (into a `Directive{Name:"pubsub",…}`), *then* hand it off to your `parsePubSub` callback. From there you can stash whatever metadata or AST annotations you need so that the code generator later emits the NATS wiring automatically—without further edits to Encore’s built‐ins.
