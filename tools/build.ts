import { exec as _exec } from "child_process"
import glob from "fast-glob"
import * as path from "path"
import * as util from "util"
import * as assert from "assert"
import * as fs from "fs"

const exec = util.promisify(_exec);

async function build(workDir: string, filename: string, platform: "win") {
  const outDir = workDir // path.join(workDir, "build"); mkdirSync(outDir)

  switch (platform) {
    case "win": {
      const HEADER_PATH = path.join(workDir, `${filename}.h`)
      const DEF_PATH = path.join(outDir, `${filename}.def`)
      const [VS_TOOLS_PATH] = await glob('**/bin/Hostx64/x64', {
        cwd: "C:/Program Files (x86)/Microsoft Visual Studio",
        onlyDirectories: true,
        absolute: true,
      })
      if (!VS_TOOLS_PATH) throw new Error('VS_TOOLS_PATH not found')
      const VS_TOOLS_PATH_ENV = {
        ...process.env,
        Path: VS_TOOLS_PATH + path.delimiter + process.env.Path
      }

      // Step 1
      const goResult = await exec(`go build -buildmode c-shared -o ${filename}.dll`, {
        cwd: workDir
      })
      assert.equal(goResult.stderr, "")
      
      // Step 2
      const header = fs.readFileSync(HEADER_PATH, "utf8")
      const editedHeader = header.replace("__SIZE_TYPE__", "size_t").replace(/^.* _Complex .*$/gm, "// $&")
      fs.writeFileSync(HEADER_PATH, editedHeader, "utf8")

      // Step 3
      const exports = await exec(`dumpbin /exports ${filename}.dll`, {
        env: VS_TOOLS_PATH_ENV,
        maxBuffer: 10 * 1024 * 1024,
        cwd: workDir
      })
      assert.equal(exports.stderr, "")
      const defFile = [`LIBRARY ${filename}`, "EXPORTS"]
      const exportsDef = exports.stdout
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
      fs.writeFileSync(DEF_PATH, defFile.join("\r\n"), "utf8")
      await exec(`lib /def:${filename}.def /out:${filename}.lib /machine:X64`, {
        env: VS_TOOLS_PATH_ENV,
        cwd: workDir
      })
      break
    }

    default:
      throw new Error("Unsupported yet!")
  }
}

build("./sample/go", "request", "win")
