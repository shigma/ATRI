import * as path from "path"
import { setCwd, step, exec, fs } from './utils'

export default async function compile(entry: string) {
  const entryDir = path.dirname(entry)
  const filename = path.basename(entry)

  setCwd(entryDir)

  // build source code
  await exec(`go build -buildmode c-shared -o ${filename}.dll`)

  // patch cgo header
  await step('patch cgo header', async () => {
    const HEADER_PATH = path.join(entryDir, `${filename}.h`)
    const header = await fs.readFile(HEADER_PATH, "utf8")
    const editedHeader = header
      .replace("__SIZE_TYPE__", "size_t")
      .replace(/^.* _Complex .*$/gm, "// $&")
    await fs.writeFile(HEADER_PATH, editedHeader, "utf8")
  })
}

if (require.main.filename === __filename) {
  compile(process.argv[2])
}
