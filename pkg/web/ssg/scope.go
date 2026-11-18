package ssg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"path"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func scopeFragment(name, body string) (template.HTML, error) {
	frag, err := html.ParseFragment(strings.NewReader(body), &html.Node{
		Type:     html.ElementNode,
		Data:     "body",
		DataAtom: atom.Body,
	})
	if err != nil {
		return "", err
	}
	target := firstElement(frag)
	if target == nil {
		return template.HTML(body), nil
	}
	for _, a := range target.Attr {
		if a.Key == "data-kiw-component" {
			return template.HTML(body), nil
		}
	}
	target.Attr = append(target.Attr, html.Attribute{Key: "data-kiw-component", Val: name})
	var buf bytes.Buffer
	for _, n := range frag {
		if err := html.Render(&buf, n); err != nil {
			return "", err
		}
	}
	return template.HTML(buf.String()), nil
}

func scopeDocument(name, doc string) (template.HTML, error) {
	root, err := html.Parse(strings.NewReader(doc))
	if err != nil {
		return "", err
	}
	for n := root.FirstChild; n != nil; n = n.NextSibling {
		if n.Type == html.ElementNode && n.Data == "html" {
			n.Attr = append(n.Attr, html.Attribute{Key: "data-kiw-layout", Val: name})
			break
		}
	}
	var buf bytes.Buffer
	if err := html.Render(&buf, root); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

func firstElement(frag []*html.Node) *html.Node {
	for _, n := range frag {
		if n.Type == html.ElementNode {
			return n
		}
	}
	return nil
}

// injectDataProps serializes props to a data-props attribute on the first
// element of a rendered fragment, giving client-side frameworks a stable,
// opt-in hydration payload.
func injectDataProps(fragHTML template.HTML, props any) (template.HTML, error) {
	data, err := json.Marshal(props)
	if err != nil {
		return "", fmt.Errorf("ssg: encode page props: %w", err)
	}
	frag, err := html.ParseFragment(strings.NewReader(string(fragHTML)), &html.Node{
		Type:     html.ElementNode,
		Data:     "body",
		DataAtom: atom.Body,
	})
	if err != nil {
		return "", err
	}
	target := firstElement(frag)
	if target == nil {
		return fragHTML, nil
	}
	target.Attr = append(target.Attr, html.Attribute{Key: "data-props", Val: string(data)})
	var buf bytes.Buffer
	for _, n := range frag {
		if err := html.Render(&buf, n); err != nil {
			return "", err
		}
	}
	return template.HTML(buf.String()), nil
}

// injectWasmLoader appends the Go WebAssembly glue and initialization script before </body>.
func (s *Site) injectWasmLoader(doc string) string {
	jsURL := s.wasm.JSURL
	if jsURL == "" {
		jsURL = s.asset("wasm_exec.js")
	}
	wasmURL := s.wasm.WASMURL
	if wasmURL == "" {
		wasmURL = s.asset("runtime.wasm")
	}

	root, err := html.Parse(strings.NewReader(doc))
	if err != nil {
		return doc
	}
	bodyNode := findElement(root, "body")
	if bodyNode == nil {
		return doc
	}
	referenced := referencedAssets(root)

	// Inject wasm_exec.js if not already present
	if !referenced[jsURL] && !referenced[path.Base(jsURL)] {
		bodyNode.AppendChild(&html.Node{
			Type:     html.ElementNode,
			Data:     "script",
			DataAtom: atom.Script,
			Attr:     []html.Attribute{{Key: "src", Val: jsURL}},
		})
	}

	loaderScript := fmt.Sprintf(`if (!WebAssembly.instantiateStreaming) {
  WebAssembly.instantiateStreaming = async (resp, importObject) => {
    const source = await (await resp).arrayBuffer();
    return await WebAssembly.instantiate(source, importObject);
  };
}
const go = new Go();
WebAssembly.instantiateStreaming(fetch("%s"), go.importObject).then((result) => {
  go.run(result.instance);
});`, wasmURL)

	scriptNode := &html.Node{
		Type:     html.ElementNode,
		Data:     "script",
		DataAtom: atom.Script,
	}
	scriptNode.AppendChild(&html.Node{
		Type: html.TextNode,
		Data: loaderScript,
	})
	bodyNode.AppendChild(scriptNode)

	var buf bytes.Buffer
	if err := html.Render(&buf, root); err != nil {
		return doc
	}
	return buf.String()
}

// injectScripts inserts <script src> tags before the closing </body> tag,
// the documented mount script slot, which is empty by default.
func injectScripts(body string, scripts []string) string {
	var sb strings.Builder
	for _, src := range scripts {
		sb.WriteString(`<script src="`)
		sb.WriteString(src)
		sb.WriteString(`"></script>`)
	}
	sb.WriteString("</body>")
	idx := strings.LastIndex(body, "</body>")
	if idx < 0 {
		return body + sb.String()
	}
	return body[:idx] + sb.String()
}
