import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App'

const operations = [
  { name: 'add', arity: 'binary', abbreviation: '+', shortcut: '+' },
  { name: 'divide', arity: 'binary', abbreviation: '/', shortcut: '/' },
  { name: 'exponentiate', arity: 'binary', abbreviation: '^', shortcut: '^' },
  { name: 'factorial', arity: 'unary', abbreviation: '!', shortcut: '!' },
  { name: 'modulus', arity: 'binary', abbreviation: '%', shortcut: '%' },
  { name: 'multiply', arity: 'binary', abbreviation: '*', shortcut: '*' },
  { name: 'square_root', arity: 'unary', abbreviation: 'sqrt' },
  { name: 'subtract', arity: 'binary', abbreviation: '-', shortcut: '-' },
]

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function setupFetch(options: {
  catalogResponse?: Response | Promise<Response>
  calculationResponses?: Response[]
} = {}) {
  const postRequests: Array<{ url: string; init?: RequestInit }> = []
  let calculationIndex = 0

  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input)
    if (init?.method === 'POST') {
      postRequests.push({ url, init })
      const response =
        options.calculationResponses?.[calculationIndex] ??
        jsonResponse({ result: 0, error: '' })
      calculationIndex += 1
      return response
    }

    return options.catalogResponse ?? jsonResponse({ operations })
  })
  vi.stubGlobal('fetch', fetchMock)
  return { fetchMock, postRequests }
}

function renderCalculator() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>,
  )
}

function display() {
  return screen.getByLabelText('Display')
}

function readRequestBody(request: { init?: RequestInit }) {
  return JSON.parse(String(request.init?.body)) as Record<string, unknown>
}

describe('calculator rendering and loading', () => {
  it('shows loading while requesting the catalog', async () => {
    let resolveCatalog: ((response: Response) => void) | undefined
    const catalogResponse = new Promise<Response>((resolve) => {
      resolveCatalog = resolve
    })
    setupFetch({ catalogResponse })
    renderCalculator()

    expect(screen.getByRole('status')).toHaveTextContent('Loading operations')

    resolveCatalog?.(jsonResponse({ operations }))
    expect(await screen.findByRole('button', { name: 'square_root, unary' })).toBeVisible()
  })

  it('renders operations fetched from the API, including optional shortcuts', async () => {
    const { fetchMock } = setupFetch()
    renderCalculator()

    const operationPanel = await screen.findByLabelText('Operations')
    expect(within(operationPanel).getAllByRole('button')).toHaveLength(9)
    expect(within(operationPanel).getByRole('button', { name: 'add, binary' })).toHaveTextContent('+')
    expect(within(operationPanel).getByRole('button', { name: 'square_root, unary' })).toHaveTextContent('sqrt')
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(fetchMock).toHaveBeenCalledWith('/api/', undefined)
  })

  it('reports a catalog HTTP error instead of showing the keypad', async () => {
    setupFetch({ catalogResponse: jsonResponse({ error: 'unavailable' }, 503) })
    renderCalculator()

    expect(await screen.findByRole('alert')).toHaveTextContent('Unable to load operations')
    expect(screen.queryByRole('button', { name: 'add, binary' })).not.toBeInTheDocument()
  })

  it('reports invalid catalog metadata', async () => {
    setupFetch({ catalogResponse: jsonResponse({ operations: [{ name: 'add' }] }) })
    renderCalculator()

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'The operations response contains invalid metadata',
    )
  })
})

describe('calculator entry and controls', () => {
  beforeEach(() => {
    setupFetch()
  })

  it('accepts digits and one decimal point, deletes a digit, and clears the display', async () => {
    const user = userEvent.setup()
    renderCalculator()
    await screen.findByRole('button', { name: 'add, binary' })

    await user.click(screen.getByRole('button', { name: '.' }))
    expect(display()).toHaveTextContent('0.')
    await user.click(screen.getByRole('button', { name: 'C' }))
    await user.click(screen.getByRole('button', { name: '1' }))
    await user.click(screen.getByRole('button', { name: '2' }))
    await user.click(screen.getByRole('button', { name: '.' }))
    await user.click(screen.getByRole('button', { name: '3' }))
    await user.click(screen.getByRole('button', { name: '.' }))
    expect(display()).toHaveTextContent('12.3')

    await user.click(screen.getByRole('button', { name: 'Backspace' }))
    expect(display()).toHaveTextContent('12.')
    await user.click(screen.getByRole('button', { name: 'C' }))
    expect(display()).toHaveTextContent('0')

    await user.click(screen.getByRole('button', { name: '0' }))
    await user.click(screen.getByRole('button', { name: '4' }))
    expect(display()).toHaveTextContent('4')
  })

  it('does not execute a binary operation when equals is pressed without a second operand', async () => {
    const user = userEvent.setup()
    const { postRequests } = setupFetch()
    renderCalculator()
    await screen.findByRole('button', { name: 'add, binary' })

    await user.click(screen.getByRole('button', { name: '1' }))
    await user.click(screen.getByRole('button', { name: '0' }))
    await user.click(screen.getByRole('button', { name: 'add, binary' }))
    await user.click(screen.getByRole('button', { name: 'Equals' }))

    expect(display()).toHaveTextContent('10')
    expect(postRequests).toHaveLength(0)
  })

  it('replaces a pending binary operation and lets clear reset the pending entry', async () => {
    const user = userEvent.setup()
    const { postRequests } = setupFetch()
    renderCalculator()
    await screen.findByRole('button', { name: 'add, binary' })

    await user.click(screen.getByRole('button', { name: '7' }))
    await user.click(screen.getByRole('button', { name: 'add, binary' }))
    await user.click(screen.getByRole('button', { name: 'subtract, binary' }))
    await user.click(screen.getByRole('button', { name: 'C' }))
    await user.click(screen.getByRole('button', { name: '2' }))
    await user.click(screen.getByRole('button', { name: 'Equals' }))

    expect(display()).toHaveTextContent('2')
    expect(postRequests).toHaveLength(0)
  })

  it('backspaces an empty entry to zero and edits the pending first operand', async () => {
    const user = userEvent.setup()
    const { postRequests } = setupFetch({
      calculationResponses: [jsonResponse({ result: 3, error: '' })],
    })
    renderCalculator()
    await screen.findByRole('button', { name: 'add, binary' })

    await user.click(screen.getByRole('button', { name: 'Backspace' }))
    expect(display()).toHaveTextContent('0')

    await user.click(screen.getByRole('button', { name: '1' }))
    await user.click(screen.getByRole('button', { name: '2' }))
    await user.click(screen.getByRole('button', { name: 'add, binary' }))
    await user.click(screen.getByRole('button', { name: 'Backspace' }))
    await user.click(screen.getByRole('button', { name: '2' }))
    await user.click(screen.getByRole('button', { name: 'Equals' }))

    await waitFor(() => expect(display()).toHaveTextContent('3'))
    expect(readRequestBody(postRequests[0])).toEqual({
      operation: 'add',
      operand_a: 1,
      operand_b: 2,
    })
  })

  it('uses the operation shortcut and Enter key to submit a binary calculation', async () => {
    const { postRequests } = setupFetch({
      calculationResponses: [jsonResponse({ result: 16, error: '' })],
    })
    renderCalculator()
    await screen.findByRole('button', { name: 'add, binary' })

    for (const key of ['1', '0']) fireEvent.keyDown(window, { key })
    fireEvent.keyDown(window, { key: '+' })
    for (const key of ['6']) fireEvent.keyDown(window, { key })
    fireEvent.keyDown(window, { key: 'Enter' })

    await waitFor(() => expect(display()).toHaveTextContent('16'))
    expect(postRequests).toHaveLength(1)
    expect(postRequests[0].url).toBe('/api/')
    expect(readRequestBody(postRequests[0])).toEqual({
      operation: 'add',
      operand_a: 10,
      operand_b: 6,
    })
  })

  it('supports decimal and backspace keyboard input and ignores shortcut-less operations', async () => {
    const { postRequests } = setupFetch()
    renderCalculator()
    await screen.findByRole('button', { name: 'square_root, unary' })

    fireEvent.keyDown(window, { key: '1' })
    fireEvent.keyDown(window, { key: '.' })
    fireEvent.keyDown(window, { key: '2' })
    fireEvent.keyDown(window, { key: 'Backspace' })
    expect(display()).toHaveTextContent('1.')

    fireEvent.keyDown(window, { key: 's' })
    expect(display()).toHaveTextContent('1.')
    expect(postRequests).toHaveLength(0)
  })

  it('does not capture keyboard input from editable elements', async () => {
    renderCalculator()
    await screen.findByRole('button', { name: 'add, binary' })
    const input = document.createElement('input')
    document.body.append(input)

    fireEvent.keyDown(input, { key: '9' })
    expect(display()).toHaveTextContent('0')
    input.remove()
  })

  it('executes a unary operation immediately with one operand', async () => {
    const user = userEvent.setup()
    const { postRequests } = setupFetch({
      calculationResponses: [jsonResponse({ result: 120, error: '' })],
    })
    renderCalculator()
    await screen.findByRole('button', { name: 'factorial, unary' })

    await user.click(screen.getByRole('button', { name: '5' }))
    await user.click(screen.getByRole('button', { name: 'factorial, unary' }))

    await waitFor(() => expect(display()).toHaveTextContent('120'))
    expect(readRequestBody(postRequests[0])).toEqual({
      operation: 'factorial',
      operand_a: 5,
    })
    expect(readRequestBody(postRequests[0])).not.toHaveProperty('operand_b')
  })

  it('executes a pending binary operation before the next binary operation', async () => {
    const user = userEvent.setup()
    const { postRequests } = setupFetch({
      calculationResponses: [
        jsonResponse({ result: 16, error: '' }),
        jsonResponse({ result: 32, error: '' }),
      ],
    })
    renderCalculator()
    await screen.findByRole('button', { name: 'add, binary' })

    for (const digit of ['1', '0']) await user.click(screen.getByRole('button', { name: digit }))
    await user.click(screen.getByRole('button', { name: 'add, binary' }))
    await user.click(screen.getByRole('button', { name: '6' }))
    await user.click(screen.getByRole('button', { name: 'multiply, binary' }))
    await waitFor(() => expect(display()).toHaveTextContent('16'))

    await user.click(screen.getByRole('button', { name: '2' }))
    await user.click(screen.getByRole('button', { name: 'Equals' }))
    await waitFor(() => expect(display()).toHaveTextContent('32'))

    expect(readRequestBody(postRequests[0])).toEqual({
      operation: 'add',
      operand_a: 10,
      operand_b: 6,
    })
    expect(readRequestBody(postRequests[1])).toEqual({
      operation: 'multiply',
      operand_a: 16,
      operand_b: 2,
    })
  })

  it('runs a unary operation on the pending first operand when no second operand was entered', async () => {
    const user = userEvent.setup()
    const { postRequests } = setupFetch({
      calculationResponses: [jsonResponse({ result: 4, error: '' })],
    })
    renderCalculator()
    await screen.findByRole('button', { name: 'add, binary' })

    await user.click(screen.getByRole('button', { name: '1' }))
    await user.click(screen.getByRole('button', { name: '6' }))
    await user.click(screen.getByRole('button', { name: 'add, binary' }))
    await user.click(screen.getByRole('button', { name: 'square_root, unary' }))

    await waitFor(() => expect(display()).toHaveTextContent('4'))
    expect(postRequests).toHaveLength(1)
    expect(readRequestBody(postRequests[0])).toEqual({
      operation: 'square_root',
      operand_a: 16,
    })
  })

  it('completes a pending binary operation before immediately running a unary operation', async () => {
    const user = userEvent.setup()
    const { postRequests } = setupFetch({
      calculationResponses: [
        jsonResponse({ result: 5, error: '' }),
        jsonResponse({ result: 120, error: '' }),
      ],
    })
    renderCalculator()
    await screen.findByRole('button', { name: 'add, binary' })

    await user.click(screen.getByRole('button', { name: '2' }))
    await user.click(screen.getByRole('button', { name: 'add, binary' }))
    await user.click(screen.getByRole('button', { name: '3' }))
    await user.click(screen.getByRole('button', { name: 'factorial, unary' }))

    await waitFor(() => expect(display()).toHaveTextContent('120'))
    expect(readRequestBody(postRequests[0])).toEqual({
      operation: 'add',
      operand_a: 2,
      operand_b: 3,
    })
    expect(readRequestBody(postRequests[1])).toEqual({
      operation: 'factorial',
      operand_a: 5,
    })
  })

  it('shows Err on calculation errors and replaces it with new input', async () => {
    const user = userEvent.setup()
    const { postRequests } = setupFetch({
      calculationResponses: [jsonResponse({ result: 'err', error: 'divisor cannot be zero' })],
    })
    renderCalculator()
    await screen.findByRole('button', { name: 'divide, binary' })

    await user.click(screen.getByRole('button', { name: '8' }))
    await user.click(screen.getByRole('button', { name: 'divide, binary' }))
    await user.click(screen.getByRole('button', { name: '0' }))
    await user.click(screen.getByRole('button', { name: 'Equals' }))

    await waitFor(() => expect(display()).toHaveTextContent('Err'))
    expect(screen.getByText('divisor cannot be zero')).toBeVisible()

    await user.click(screen.getByRole('button', { name: 'square_root, unary' }))
    expect(postRequests).toHaveLength(1)

    await user.click(screen.getByRole('button', { name: '2' }))
    expect(display()).toHaveTextContent('2')
  })
})
