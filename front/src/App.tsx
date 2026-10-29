import { useCallback, useEffect, useRef, useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { calculate, type CalculationRequest, type Operation } from './api'
import { useOperations } from './hooks/useOperations'
import './App.css'

interface PendingBinaryOperation {
  operation: Operation
  operandA: number
  operandAText: string
}

const EMPTY_OPERATIONS: Operation[] = []

function formatNumber(value: number): string {
  return Object.is(value, -0) ? '0' : String(value)
}

function getOperand(value: string): number | null {
  const operand = Number(value)
  return value.trim() !== '' && Number.isFinite(operand) ? operand : null
}

function App() {
  const operationsQuery = useOperations()
  const operations = operationsQuery.data ?? EMPTY_OPERATIONS
  const { isPending: mutationPending, mutateAsync } = useMutation({ mutationFn: calculate })
  const [display, setDisplay] = useState('0')
  const [entryStarted, setEntryStarted] = useState(false)
  const [pendingOperation, setPendingOperation] = useState<PendingBinaryOperation | null>(null)
  const [calculationPending, setCalculationPending] = useState(false)
  const [statusMessage, setStatusMessage] = useState('')
  const inFlight = useRef(false)
  const displayRef = useRef<HTMLOutputElement>(null)
  const isBusy = mutationPending || calculationPending

  useEffect(() => {
    const displayElement = displayRef.current
    if (displayElement) {
      displayElement.scrollLeft = displayElement.scrollWidth
    }
  }, [display])

  const executeCalculation = useCallback(async (
    request: CalculationRequest,
    onSuccess?: (result: number) => void,
  ): Promise<number | null> => {
    if (inFlight.current) return null

    inFlight.current = true
    setCalculationPending(true)
    setStatusMessage('')
    try {
      const result = await mutateAsync(request)

      setDisplay(formatNumber(result))

      setEntryStarted(false)
      setPendingOperation(null)

      onSuccess?.(result)

      return result
    } catch (error) {
      setDisplay('Err')
      setEntryStarted(false)
      setPendingOperation(null)

      setStatusMessage(error instanceof Error ? error.message : 'Calculation failed')

      return null
    } finally {
      inFlight.current = false
      setCalculationPending(false)
    }
  }, [mutateAsync])

  const enterDigit = useCallback((digit: string) => {
    if (isBusy || inFlight.current) return

    setStatusMessage('')
    if (display === 'Err' || !entryStarted) {
      setDisplay(digit)
      setEntryStarted(true)
      return
    }

    if (display === '0') {
      setDisplay(digit)
      return
    }

    setDisplay((current) => current + digit)
  }, [display, entryStarted, isBusy])

  const enterDecimalPoint = useCallback(() => {
    if (isBusy || inFlight.current) return

    setStatusMessage('')
    if (display === 'Err' || !entryStarted) {
      setDisplay('0.')
      setEntryStarted(true)
      return
    }
    if (!display.includes('.')) {
      setDisplay((current) => current + '.')
    }
  }, [display, entryStarted, isBusy])

  const clearCalculator = useCallback(() => {
    if (isBusy || inFlight.current) return
    setDisplay('0')
    setEntryStarted(false)
    setPendingOperation(null)
    setStatusMessage('')
  }, [isBusy])

  const deleteLastCharacter = useCallback(() => {
    if (isBusy || inFlight.current || display === 'Err') return

    setStatusMessage('')
    const updatedDisplay = display.slice(0, -1)
    const nextDisplay = updatedDisplay === '' || updatedDisplay === '-' ? '0' : updatedDisplay
    setDisplay(nextDisplay)

    if (pendingOperation && !entryStarted) {
      const operandA = getOperand(nextDisplay)
      if (operandA !== null) {
        setPendingOperation({ ...pendingOperation, operandA, operandAText: nextDisplay })
      }
    }
  }, [display, entryStarted, isBusy, pendingOperation])

  const selectOperation = useCallback(async (operation: Operation) => {
    if (isBusy || inFlight.current || display === 'Err') return
    setStatusMessage('')

    const currentOperand = getOperand(display)
    if (currentOperand === null) return

    if (operation.arity === 'binary') {
      if (pendingOperation && entryStarted) {
        const secondOperand = currentOperand
        await executeCalculation(
          {
            operation: pendingOperation.operation.name,
            operand_a: pendingOperation.operandA,
            operand_b: secondOperand,
          },
          (result) => {
            const resultText = formatNumber(result)
            setPendingOperation({
              operation,
              operandA: result,
              operandAText: resultText,
            })
          },
        )
        return
      }

      const operandA = pendingOperation?.operandA ?? currentOperand
      const operandAText = pendingOperation?.operandAText ?? display
      setPendingOperation({ operation, operandA, operandAText })
      setDisplay(operandAText)
      setEntryStarted(false)
      return
    }

    if (pendingOperation && entryStarted) {
      const secondOperand = currentOperand
      const result = await executeCalculation({
        operation: pendingOperation.operation.name,
        operand_a: pendingOperation.operandA,
        operand_b: secondOperand,
      })
      if (result !== null) {
        await executeCalculation({ operation: operation.name, operand_a: result })
      }
      return
    }

    const operandA = pendingOperation?.operandA ?? currentOperand
    setPendingOperation(null)
    await executeCalculation({ operation: operation.name, operand_a: operandA })
  }, [display, entryStarted, executeCalculation, isBusy, pendingOperation])

  const executeEquals = useCallback(() => {
    if (isBusy || inFlight.current || display === 'Err' || !pendingOperation) return

    const secondOperand = entryStarted ? getOperand(display) : null
    if (secondOperand === null) {
      setDisplay(pendingOperation.operandAText)
      return
    }

    void executeCalculation({
      operation: pendingOperation.operation.name,
      operand_a: pendingOperation.operandA,
      operand_b: secondOperand,
    })
  }, [display, entryStarted, executeCalculation, isBusy, pendingOperation])

  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      const target = event.target
      if (
        target instanceof HTMLElement &&
        (target.isContentEditable ||
          target instanceof HTMLInputElement ||
          target instanceof HTMLTextAreaElement ||
          target instanceof HTMLSelectElement)
      ) {
        return
      }

      if (/^[0-9]$/.test(event.key)) {
        event.preventDefault()
        enterDigit(event.key)
        return
      }
      if (event.key === '.') {
        event.preventDefault()
        enterDecimalPoint()
        return
      }
      if (event.key === 'Enter') {
        event.preventDefault()
        executeEquals()
        return
      }
      if (event.key === 'Backspace') {
        event.preventDefault()
        deleteLastCharacter()
        return
      }

      const matchingOperation = operations.find(
        (operation) => operation.shortcut && operation.shortcut === event.key,
      )
      if (matchingOperation) {
        event.preventDefault()
        void selectOperation(matchingOperation)
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [
    operations,
    deleteLastCharacter,
    enterDecimalPoint,
    enterDigit,
    executeEquals,
    selectOperation,
  ])

  if (operationsQuery.isPending) {
    return (
      <main className="app-shell">
        <section className="calculator" aria-busy="true">
          <h1 className="app-title">Calculator</h1>
          <p className="loading-message" role="status">Loading operations…</p>
        </section>
      </main>
    )
  }

  if (operationsQuery.isError) {
    return (
      <main className="app-shell">
        <section className="calculator">
          <h1 className="app-title">Calculator</h1>
          <p className="catalog-error" role="alert">
            Unable to load operations: {operationsQuery.error.message}
          </p>
        </section>
      </main>
    )
  }

  return (
    <main className="app-shell">
      <section className="calculator" aria-label="Calculator">
        <h1 className="app-title">Calculator</h1>
        <output
          ref={displayRef}
          className={`display${display === 'Err' ? ' display-error' : ''}`}
          aria-label="Display"
          aria-live="polite"
          aria-atomic="true"
          tabIndex={0}
        >
          <span className="display-value">{display}</span>
        </output>
        <div className="keypad">
          <div className="utility-keys">
            <button type="button" className="key key-utility" onClick={clearCalculator} disabled={isBusy}>
              C
            </button>
            <button
              type="button"
              className="key key-utility"
              onClick={deleteLastCharacter}
              disabled={isBusy}
              aria-label="Backspace"
            >
              ⌫
            </button>
          </div>
          <div className="digit-keys" aria-label="Digits">
            {['7', '8', '9', '4', '5', '6', '1', '2', '3', '0', '.'].map((key) => (
              <button
                type="button"
                className={`key${key === '0' ? ' key-zero' : ''}`}
                key={key}
                onClick={() => (key === '.' ? enterDecimalPoint() : enterDigit(key))}
                disabled={isBusy}
              >
                {key}
              </button>
            ))}
          </div>
          <div className="operation-keys" aria-label="Operations">
            {[...operations]
              .sort((a, b) => (a.arity === b.arity ? 0 : a.arity === 'binary' ? -1 : 1))
              .map((operation) => (
              <button
                type="button"
                className={`key key-operation key-${operation.arity}`}
                key={operation.name}
                onClick={() => void selectOperation(operation)}
                disabled={isBusy}
                aria-label={`${operation.name}, ${operation.arity}`}
              >
                {operation.abbreviation}
              </button>
              ))}
            <button
              type="button"
              className="key key-equals"
              onClick={executeEquals}
              disabled={isBusy}
              aria-label="Equals"
            >
              =
            </button>
          </div>
        </div>
        <p className="status-message" role="status" aria-live="polite">
          {isBusy ? 'Calculating…' : statusMessage}
        </p>
      </section>
    </main>
  )
}

export default App
