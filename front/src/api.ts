export type OperationArity = 'binary' | 'unary'

export interface Operation {
  name: string
  arity: OperationArity
  abbreviation: string
  shortcut?: string
}

export interface CalculationRequest {
  operation: string
  operand_a: number
  operand_b?: number
}

const apiBaseUrl = (import.meta.env.VITE_API_BASE_URL ?? '/api').replace(/\/+$/, '')

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

async function getResponse(url: string, init?: RequestInit): Promise<Response> {
  const response = await fetch(url, init)
  if (!response.ok) {
    throw new Error(`Request failed (${response.status} ${response.statusText})`)
  }
  return response
}

export async function fetchOperations(): Promise<Operation[]> {
  const response = await getResponse(`${apiBaseUrl}/`)
  const body: unknown = await response.json()

  if (!isRecord(body) || !Array.isArray(body.operations)) {
    throw new Error('The operations response is invalid')
  }

  return body.operations.map((value: unknown) => {
    if (
      !isRecord(value) ||
      typeof value.name !== 'string' ||
      (value.arity !== 'binary' && value.arity !== 'unary') ||
      typeof value.abbreviation !== 'string' ||
      (value.shortcut !== undefined && typeof value.shortcut !== 'string')
    ) {
      throw new Error('The operations response contains invalid metadata')
    }

    return {
      name: value.name,
      arity: value.arity,
      abbreviation: value.abbreviation,
      ...(typeof value.shortcut === 'string' ? { shortcut: value.shortcut } : {}),
    }
  })
}

export async function calculate(request: CalculationRequest): Promise<number> {
  const response = await getResponse(`${apiBaseUrl}/`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request),
  })
  const body: unknown = await response.json()

  if (!isRecord(body) || typeof body.error !== 'string') {
    throw new Error('The calculation response is invalid')
  }
  if (body.error !== '') {
    throw new Error(body.error)
  }
  if (typeof body.result !== 'number' || !Number.isFinite(body.result)) {
    throw new Error('The calculation result is invalid')
  }

  return body.result
}
