package cmd

import "testing"

func TestResolveFunctionFile(t *testing.T) {
	canonical, path, err := resolveFunctionFile("demo:world/load")
	if err != nil {
		t.Fatal(err)
	}
	if canonical != "demo:world/load" {
		t.Fatalf("canonical name = %q", canonical)
	}
	if path != "functions/demo/world/load.mcfunction" {
		t.Fatalf("path = %q", path)
	}
	canonical, path, err = resolveFunctionFile("welcome")
	if err != nil {
		t.Fatal(err)
	}
	if canonical != "minecraft:welcome" || path != "functions/minecraft/welcome.mcfunction" {
		t.Fatalf("default namespace result = %q, %q", canonical, path)
	}
}

func TestResolveFunctionFileRejectsTraversal(t *testing.T) {
	for _, name := range []string{
		"../secret",
		"demo:../secret",
		"demo:a/../../secret",
		"demo:/absolute",
		"demo:a\\secret",
		"Demo:upper",
		"demo:",
	} {
		if _, _, err := resolveFunctionFile(name); err == nil {
			t.Errorf("expected %q to be rejected", name)
		}
	}
}

func TestSplitFunctionCommand(t *testing.T) {
	name, args := splitFunctionCommand("give @s minecraft:stone 2")
	if name != "give" {
		t.Fatalf("command name = %q", name)
	}
	if args != "@s minecraft:stone 2" {
		t.Fatalf("args = %q", args)
	}
}
