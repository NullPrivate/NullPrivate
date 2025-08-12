## Project Overview

This project is **NullPrivate**, a fork of the popular network-wide ad-blocker **AdGuardHome**. It functions as a DNS server, blocking ads and trackers on all devices across a network.

The backend is written in **Go**, and the frontend is a **React**-based web dashboard.

## Building and Running

The project uses a `Makefile` to streamline the build and development process.

**Key Commands:**

*   **Install Dependencies:**
    ```bash
    make deps
    ```

*   **Build (Frontend & Backend):**
    ```bash
    make build
    ```

*   **Run Tests:**
    ```bash
    make test
    ```

*   **Run Linters:**
    ```bash
    make lint
    ```

*   **Run the Application (Development):**
    After building the application with `make build`, you can run it using the command from the `README.md`:
    ```bash
    ./NullPrivate -c ./AdGuardHome.yaml -w ./data --web-addr 0.0.0.0:34020 --local-frontend --no-check-update --verbose
    ```

## Development Conventions

*   **Backend:** Go code is located in the `internal` directory. Dependencies are managed with `go.mod`.
*   **Frontend:** The React application source is in `client/src`. It uses `npm` for package management and `webpack` for bundling.
*   **Contributions:** The `CONTRIBUTING.md` file likely contains information about contribution guidelines.
