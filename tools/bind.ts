import { step, exec, fs, spawn, setCwd } from "./utils"
import * as path from "path"
import glob from "fast-glob"

export default async function bind(entry: string, bindingDir: string) {
  const entryDir = path.dirname(entry)
  const filename = path.basename(entry)

  // find vs tools
  await step('find vs tools', async () => {
    const dirs = await glob('**/bin/Hostx64/x64', {
      cwd: "C:/Program Files (x86)/Microsoft Visual Studio",
      onlyDirectories: true,
      absolute: true,
    })
    // always use the latest version
    const [VS_TOOLS_PATH] = dirs.reverse()
    if (!VS_TOOLS_PATH) throw new Error('vs tools not found')
    process.env.PATH = VS_TOOLS_PATH + path.delimiter + process.env.PATH
  })

  setCwd(entryDir)

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
        return parts.length === 4 ? parts[3] : []
      })
    defFile.push(...exportsDef)
    const DEF_PATH = path.join(entryDir, `${filename}.def`)
    await fs.writeFile(DEF_PATH, defFile.join("\r\n"), "utf8")
  })

  await exec(`lib /def:${filename}.def /out:${filename}.lib /machine:X64`)

  // node-gyp
  // rebuild = clean + configure + build
  await spawn('node-gyp.cmd rebuild', { cwd: bindingDir })

  // copy dependency
  await step('copy dependency', async () => {
    await fs.copyFile(entry + ".dll", path.join(bindingDir, 'build/release', filename + '.dll'))
  })
}

if (require.main.filename === __filename) {
  bind(process.argv[2], process.argv[3])
}
