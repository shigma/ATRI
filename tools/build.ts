import { step, exec, fs, _exec } from "./utils"
import * as path from "path"
import glob from "fast-glob"
import cgo from "./cgo"

async function build(entry: string, bindingDir: string, platform: "win") {
  const entryDir = path.dirname(entry)
  const filename = path.basename(entry)

  switch (platform) {
    case "win": {
      await cgo(entry)

      // find vs tools
      await step('find vs tools', async () => {
        const [VS_TOOLS_PATH] = await glob('**/bin/Hostx64/x64', {
          cwd: "C:/Program Files (x86)/Microsoft Visual Studio",
          onlyDirectories: true,
          absolute: true,
        })
        if (!VS_TOOLS_PATH) throw new Error('vs tools not found')
        process.env.PATH = VS_TOOLS_PATH + path.delimiter + process.env.PATH
      })

      // generate lib from dll
      const exports = await exec(`dumpbin /exports ${filename}.dll`, {
        maxBuffer: 10 * 1024 * 1024,
      })

      await step('write def file', async () => {
        const defFile = [`LIBRARY ${filename}`, "EXPORTS"]
        const exportsDef = exports
          .split("\r\n")
          .slice(19) // starts from 20th line
          .flatMap(line => {
            const parts = line.trim().split(/ +/)
            if (parts.length === 4) {
              return parts[3]
            } else {
              return []
            }
          })
        defFile.push(...exportsDef)
        const DEF_PATH = path.join(entryDir, `${filename}.def`)
        await fs.writeFile(DEF_PATH, defFile.join("\r\n"), "utf8")
      })

      await exec(`lib /def:${filename}.def /out:${filename}.lib /machine:X64`)

      // node-gyp
      // rebuild = clean + configure + build
      await step('node gyp rebuild', async () => {
        await _exec('node-gyp rebuild', { cwd: bindingDir })
      })
    
      // copy dependency
      await step('copy dependency', async () => {
        await fs.copyFile(entry + ".dll", path.join(bindingDir, 'build/release', filename + '.dll'))
      })

      break
    }

    default:
      throw new Error("Unsupported yet!")
  }
}

build("./sample/go/request", "./sample/binding", "win")
