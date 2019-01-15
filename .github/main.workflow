workflow "Build" {
  on = "push"
  resolves = "Test"
}

action "Test" {
  uses = "./.github/action/build"
}
