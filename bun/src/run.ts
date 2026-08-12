import { spawn } from 'node:child_process';

export function runBinary(binaryPath: string, args: string[]): Promise<number> {
  return new Promise((resolve, reject) => {
    const child = spawn(binaryPath, args, {
      stdio: 'inherit',
      env: { ...process.env },
    });

    child.on('error', reject);
    child.on('close', (code) => resolve(code ?? 1));
  });
}
