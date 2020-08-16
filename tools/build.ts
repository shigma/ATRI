import compile from "./compile"
import bind from "./bind"

async function build(entry: string, bindingDir: string, platform: "win") {
  await compile(entry)
  await bind(entry, bindingDir)
}

build("./src/main", "./bind", "win")
