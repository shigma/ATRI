import * as cp from "child_process"
import * as path from "path"
import * as util from "util"
import * as fs from "fs"
import glob from "fast-glob"
import ora from "ora"
import ms from "ms"

async function build(workDir: string, filename: string, platform: "win") {
  const outDir = workDir // path.join(workDir, "build"); mkdirSync(outDir)
  const spinner = ora()

  const _exec = util.promisify(cp.exec)
  async function exec (command: string, options: cp.ExecOptions = {}) {
    const timestamp = Date.now()
    spinner.start(command)
    const { stdout, stderr } = await _exec(command, {
      cwd: workDir,
      ...options,
    })
    if (stderr) {
      spinner.fail(stderr)
      process.exit(1)
    }
    spinner.text += ` (+${ms(Date.now() - timestamp)})`
    spinner.succeed()
    return stdout
  }

  switch (platform) {
    case "win": {
      const HEADER_PATH = path.join(workDir, `${filename}.h`)
      const DEF_PATH = path.join(outDir, `${filename}.def`)

      // find vs tools
      spinner.start('find vs tools')
      const [VS_TOOLS_PATH] = await glob('**/bin/Hostx64/x64', {
        cwd: "C:/Program Files (x86)/Microsoft Visual Studio",
        onlyDirectories: true,
        absolute: true,
      })
      if (!VS_TOOLS_PATH) throw new Error('vs tools not found')
      spinner.succeed(`vs tools found at: ${VS_TOOLS_PATH}`)
      process.env.PATH = VS_TOOLS_PATH + path.delimiter + process.env.PATH

      // build source code
      await exec(`go build -buildmode c-shared -o ${filename}.dll`)
  
      // patch cgo header
      spinner.start('patch cgo header')
      const header = await fs.promises.readFile(HEADER_PATH, "utf8")
      const editedHeader = header.replace("__SIZE_TYPE__", "size_t").replace(/^.* _Complex .*$/gm, "// $&")
      await fs.promises.writeFile(HEADER_PATH, editedHeader, "utf8")
      spinner.succeed()

      // generate lib from dll
      const exports = await exec(`dumpbin /exports ${filename}.dll`, {
        maxBuffer: 10 * 1024 * 1024,
      })

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
      await exec(`lib /def:${filename}.def /out:${filename}.lib /machine:X64`)
      break
    }

    default:
      throw new Error("Unsupported yet!")
  }
}

build("./sample/go", "request", "win")
