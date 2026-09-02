// exception_handling fixture: 1.0 ICP per block. Expected total for
// `exceptions`: 5.
export function exceptions(v: string): number {
  let n = 0;
  try {
    // +1  the try body
    n = JSON.parse(v);
  } catch {
    // +1  catch
    n = -1;
  } finally {
    // +1  finally
    n += 1;
  }

  try {
    // +1  the try body
    n += JSON.parse(v);
  } finally {
    // +1  finally; try/finally is 2
    n += 1;
  }

  return n;
}
