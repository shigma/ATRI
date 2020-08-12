import * as cp from "child_process"
import * as path from "path"
import * as util from "util"
import * as fs from "fs"
import glob from "fast-glob"
import chalk from "chalk"
import ora from "ora"
import ms from "ms"

async function build(entry: string, bindingDir: string, platform: "win") {
  const entryDir = path.dirname(entry)
  const filename = path.basename(entry)
  const spinner = ora()

  let timestamp: number
  function markStart(text: string) {
    timestamp = Date.now()
    spinner.start(text)
  }

  function markSucceed() {
    spinner.text += chalk.green` (+${ms(Date.now() - timestamp)})`
    spinner.succeed()
  }

  const _exec = util.promisify(cp.exec)
  async function exec (command: string, options: cp.ExecOptions = {}) {
    markStart(command)
    const { stdout, stderr } = await _exec(command, {
      cwd: entryDir,
      ...options,
    })
    if (stderr) {
      spinner.fail(stderr)
      process.exit(1)
    }
    markSucceed()
    return stdout
  }

  switch (platform) {
    case "win": {
      const HEADER_PATH = path.join(entryDir, `${filename}.h`)
      const DEF_PATH = path.join(entryDir, `${filename}.def`)

      // find vs tools
      markStart('find vs tools')
      const [VS_TOOLS_PATH] = await glob('**/bin/Hostx64/x64', {
        cwd: "C:/Program Files (x86)/Microsoft Visual Studio",
        onlyDirectories: true,
        absolute: true,
      })
      if (!VS_TOOLS_PATH) throw new Error('vs tools not found')
      process.env.PATH = VS_TOOLS_PATH + path.delimiter + process.env.PATH
      markSucceed()

      // build source code
      await exec(`go build -buildmode c-shared -o ${filename}.dll`)
  
      // patch cgo header
      markStart('patch cgo header')
      const header = await fs.promises.readFile(HEADER_PATH, "utf8")
      const editedHeader = header.replace("__SIZE_TYPE__", "size_t").replace(/^.* _Complex .*$/gm, "// $&")
      await fs.promises.writeFile(HEADER_PATH, editedHeader, "utf8")
      markSucceed()

      // generate lib from dll
      const exports = await exec(`dumpbin /exports ${filename}.dll`, {
        maxBuffer: 10 * 1024 * 1024,
      })

      markStart('write def file')
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
      await fs.promises.writeFile(DEF_PATH, defFile.join("\r\n"), "utf8")
      markSucceed()

      await exec(`lib /def:${filename}.def /out:${filename}.lib /machine:X64`)

      // node-gyp
      // rebuild = clean + configure + build
      markStart('node gyp rebuild')
      await _exec('node-gyp rebuild', { cwd: bindingDir })
      markSucceed()
    
      // copy dependency
      markStart('copy dependency')
      await fs.promises.copyFile(entry + ".dll", path.join(bindingDir, 'build/release', filename + '.dll'))
      markSucceed()

      break
    }

    default:
      throw new Error("Unsupported yet!")
  }
}

build("./sample/go/request", "./sample/binding", "win")
