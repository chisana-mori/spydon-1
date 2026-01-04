# Spydon Pipeline System Implementation Status

## 1. Overview
The Spydon Pipeline System is a stage-based orchestration engine integrated into the Robusta-Web platform. It enables the automation of operational tasks by orchestrating AWX (Ansible Tower) jobs, manual approvals, delays, and health checks.

## 2. Implemented Features

### 2.1 Backend (`apps/backend`)

#### Core Engine (`PipelineEngine`)
*   **Sequential Execution**: Executes stages in defined order using background goroutines.
*   **Breakpoint Resume (Fixed)**: Intelligent resumption of paused/failed pipelines. Logic correctly identifies and skips already `Successful` or `Skipped` stages to prevent duplicate execution (Critical fix).
*   **Parameter Management**:
    *   Supports dynamic parameter binding for stages.
    *   Validates input parameters (Required fields, Select options) before execution.
*   **Persistence**: Full state persistence using MySQL (via GORM).

#### Supported Stage Types
1.  **AWX Job (`awx_job`)**
    *   **Integration**: Direct API integration with AWX/Tower.
    *   **Features**:
        *   Template Selection (ID/Name).
        *   Parameter Injection (`extra_vars`).
        *   Dry Run (Check Mode) support.
        *   Log retrieval (Status and stdout).
    *   **Optimization**: `WaitForJob` logic optimized for immediate status checks.
2.  **Manual Gate (`manual_gate`)**
    *   **Function**: Pauses execution flow until human approval.
    *   **Actions**: Support `Resume` (Approve) via API.
3.  **Delay (`delay`)**
    *   **Function**: Time-based wait.
4.  **Pre/Post Check (`pre_check`, `post_check`)** (New)
    *   **Integration**: Prometheus HTTP API.
    *   **Logic**:
        *   Executes PromQL queries.
        *   Parses Vector results.
        *   Compares results against `Expected Value` using operators (`=`, `>`, `<`, `!=`, etc.).
5.  **Rollback**
    *   **Logic**: Triggers specific rollback stages upon failure if configured.

#### Testing
*   **Unit Tests**: Comprehensive coverage (`pipeline_engine_test.go`) using `sqlite` (in-memory) and `httptest` mocks.
    *   Covers normal flow (Start -> Job -> Check -> Success).
    *   Covers resume flow (Pause -> Resume -> Skip Completed -> Next Stage).

### 2.2 Frontend (`apps/frontend`)

#### Pipeline Management
*   **Template Editor**:
    *   **AWX Integration**: Live dropdown selection of AWX Job Templates (fetched from backend).
    *   **Parameter Definition**: UI to define input parameters (`Text`, `Select`, `MultiSelect`, `Fixed`) for template consumers.
    *   **Stage Configuration**: setup for all supported stage types.

#### Execution Interface
*   **Launch Dialog**:
    *   **Dynamic Form**: Automatically renders input form based on Template Parameter definitions.
    *   **Input Handling**: Supports default values and validation.
*   **Visual Status**: View stage-by-stage execution progress and logs.

## 3. Configuration & Data Models

### 3.1 Stage Configuration (`StageConfig`)
*   `awx_template_id`: ID of the AWX Job Template.
*   `extra_vars`: Key-value pairs passed to Ansible.
*   `parameters`: Definition of runtime parameters exposed to user.
*   `prom_query`: PromQL for checks.
*   `operator` / `expected_value`: Validation logic for checks.
*   `dry_run`: Boolean to enable Check Mode.

### 3.2 Failure Strategies
*   `abort`: Stop execution immediately on failure.
*   `pause`: Pause and wait for intervention.
*   `rollback`: Execute defined rollback stages.

## 4. Known Limitations / Future Roadmap
1.  **Manual Gate Enhancements**:
    *   `timeout_minutes` (Auto-rejection) is defined in model but logic is pending.
    *   `approver_roles` (RBAC) is defined but currently not enforced/validated by backend.
2.  **Condition Stage**:
    *   Logic for conditional branching (`models.StageTypeCondition`) is currently a TODO.
3.  **AWX Template Fallback**:
    *   Current backend relies strictly on `AWXTemplateID`. Fallback to `Name` lookup for robustness is recommended.

## 5. Summary
The pipeline system is now functional for core use cases: orchestrating Ansible jobs with pre/post health checks and manual approvals. The critical duplicate-execution bug has been resolved, and the system is covered by tests.
