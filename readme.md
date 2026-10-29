# Calculator

The calculator loads its operation catalog from the Go service and sends all calculations to that service. It does not evaluate operations in the browser.

## Run locally

Start the service from the repository's `service` directory:

```powershell
go run .
```

In another terminal, start the frontend from `front`:

```powershell
npm install
npm run dev
```

Vite proxies frontend requests from `/api/` to `http://localhost:8080/` during development.

## Tests

Run the frontend unit and integration tests with Vitest and React Testing Library:

```powershell
npm test
npm run test:coverage
```

The tests mock the API using `fetch` responses; they do not require the Go service to be running. `npm run test:watch` starts Vitest in watch mode.

For deployments where the API is not available at the same origin under `/api/`, set `VITE_API_BASE_URL` to the API base URL at build time (for example, `https://api.example.com`). The frontend requests the root endpoint beneath that base URL.

The operation list is fetched once per page load. Binary calculations run when the user presses equals or chooses a subsequent operation after entering the second operand. Unary calculations run immediately when their operation is selected.
