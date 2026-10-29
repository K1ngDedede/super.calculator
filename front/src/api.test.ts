import { afterEach, describe, expect, it, vi } from 'vitest'
import { calculate, fetchOperations } from './api'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('calculator API', () => {
  it('fetches and validates the operation catalog', async () => {
    const operations = [
      { name: 'add', arity: 'binary', abbreviation: '+', shortcut: '+' },
      { name: 'square_root', arity: 'unary', abbreviation: 'sqrt' },
    ]
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ operations }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await expect(fetchOperations()).resolves.toEqual(operations)
    expect(fetchMock).toHaveBeenCalledWith('/api/', undefined)
  })

  it('rejects invalid operation metadata and non-success HTTP responses', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn()
        .mockResolvedValueOnce(
          new Response(JSON.stringify({ operations: [{ name: 'bad' }] }), { status: 200 }),
        )
        .mockResolvedValueOnce(new Response('unavailable', { status: 503, statusText: 'Unavailable' })),
    )

    await expect(fetchOperations()).rejects.toThrow('invalid metadata')
    await expect(fetchOperations()).rejects.toThrow('Request failed (503 Unavailable)')
  })

  it('rejects malformed catalog JSON and invalid shortcut metadata', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn()
        .mockResolvedValueOnce(new Response('not json', { status: 200 }))
        .mockResolvedValueOnce(
          new Response(
            JSON.stringify({
              operations: [{ name: 'factorial', arity: 'unary', abbreviation: '!', shortcut: 1 }],
            }),
            { status: 200 },
          ),
        ),
    )

    await expect(fetchOperations()).rejects.toThrow()
    await expect(fetchOperations()).rejects.toThrow('invalid metadata')
  })

  it('posts a calculation and returns a finite result', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ result: 16, error: '' }), { status: 200 }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await expect(calculate({ operation: 'add', operand_a: 10, operand_b: 6 })).resolves.toBe(16)
    expect(fetchMock).toHaveBeenCalledWith('/api/', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ operation: 'add', operand_a: 10, operand_b: 6 }),
    })
  })

  it('rejects calculation errors, malformed response bodies, and non-finite results', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn()
        .mockResolvedValueOnce(
          new Response(JSON.stringify({ result: 'err', error: 'division failed' }), { status: 200 }),
        )
        .mockResolvedValueOnce(
          new Response(JSON.stringify({ result: 'invalid', error: '' }), { status: 200 }),
        )
        .mockResolvedValueOnce(
          new Response(JSON.stringify({ result: null, error: '' }), { status: 200 }),
        ),
    )

    const request = { operation: 'divide', operand_a: 1, operand_b: 0 }
    await expect(calculate(request)).rejects.toThrow('division failed')
    await expect(calculate(request)).rejects.toThrow('invalid')
    await expect(calculate(request)).rejects.toThrow('result is invalid')
  })

  it('rejects calculation responses without the error field', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ result: 5 }), { status: 200 }),
      ),
    )

    await expect(calculate({ operation: 'add', operand_a: 2, operand_b: 3 })).rejects.toThrow(
      'The calculation response is invalid',
    )
  })
})
