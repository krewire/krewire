package node_test

import (
	"context"
	"fmt"

	"github.com/krewire/krewire/pkg/ui/node"
)

// Example_loginPage demonstrates how a developer composes a full page
// using typed layout and primitive Go components without writing any .kiw markup.
func Example_loginPage() {
	loginPage := node.Container(
		node.ContainerProps{
			MaxWidth: "max-w-md",
			Centered: true,
			Padding:  "py-8",
		},
		node.VStack(
			node.StackProps{Gap: "gap-4"},
			node.Heading(node.HeadingProps{Level: node.Heading1}, node.Text("Sign in")),
			node.Input(node.InputProps{
				Type:        node.InputEmail,
				Name:        "email",
				Placeholder: "name@company.com",
				Required:    true,
			}),
			node.Input(node.InputProps{
				Type:        node.InputPassword,
				Name:        "password",
				Placeholder: "••••••••",
				Required:    true,
			}),
			node.Button(
				node.ButtonProps{
					Variant: node.VariantPrimary,
					Type:    "submit",
				},
				node.Text("Continue →"),
			),
		),
	)

	out, err := node.RenderToString(context.Background(), loginPage)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Verify key structural markers exist in the output
	hasContainer := len(out) > 0 && out[:4] == "<div"
	fmt.Printf("Rendered successfully: %v\n", hasContainer)
	// Output:
	// Rendered successfully: true
}

// Example_userCard demonstrates how to create a reusable composite widget
// with typed props and named slot integration.
func Example_userCard() {
	card := node.Card(
		node.CardProps{
			Title: "Developer Profile",
			Slots: node.SlotMap{
				"footer": node.Button(node.ButtonProps{Variant: node.VariantSecondary}, node.Text("Connect")),
			},
		},
		node.VStack(
			node.StackProps{Gap: "gap-2"},
			node.Text("Krewire Core Contributor"),
			node.Badge(node.BadgeProps{Variant: node.VariantSuccess}, node.Text("Active")),
		),
	)

	html, err := node.RenderToHTML(context.Background(), card)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Valid HTML generated: %v\n", len(html) > 0)
	// Output:
	// Valid HTML generated: true
}
