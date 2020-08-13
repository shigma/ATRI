import * as cp from "child_process"
import * as util from "util"
import * as _fs from "fs"
import chalk from "chalk"
import ora from "ora"
import ms from "ms"

const spinner = ora()

let workingDir = process.cwd()

export const fs = _fs.promises

export function setCwd (cwd: string) {
  workingDir = cwd
}

export function step<T>(text: string, callback: () => Promise<T>): Promise<T> {
  const timestamp = Date.now()
  spinner.start(text)
  return callback().then((result) => {
    spinner.text += chalk.green` (+${ms(Date.now() - timestamp)})`
    spinner.succeed()
    return result
  }, (err) => {
    spinner.fail(util.inspect(err))
    process.exit(1)
  })
}

const _exec = util.promisify(cp.exec)
export function exec(command: string, options: cp.ExecOptions = {}) {
  return step(command, async () => {
    const { stdout, stderr } = await _exec(command, {
      cwd: workingDir,
      ...options,
    })
    if (stderr) throw stderr
    return stdout
  })
}

export async function spawn(command: string, options: cp.SpawnOptions) {
  console.log(chalk.blue('$'), command)
  const argv = command.split(/\s+/)
  const argv0 = argv.shift()
  const child = cp.spawn(argv0, argv, {
    stdio: 'inherit',
    ...options,
  })
  return new Promise((resolve, reject) => {
    child.on('close', (code) => {
      return code ? reject() : resolve()
    })
  })
}
