# AI Teaching Assistant

AI Teaching Assistant is a full-stack system for evaluating student submissions with LLMs.

- Backend: **Go + Gin + GORM + PostgreSQL**
- AI service: **FastAPI + OpenRouter**
- Frontend: **React + Vite + Tailwind + Axios**

## Project Overview

The platform supports role-based academic workflows:

- Student and professor authentication
- Course creation and enrollment
- Assignment creation with:
  - `title`
  - `question` (visible to students)
  - `answer_key` (used internally for AI/professor only)
- Submission flows:
  - Text answer submission
  - PDF file submission
- AI evaluation with marks, feedback, and confidence
- Professor review/override of evaluations
- Student dashboard view of own submissions and evaluation history

## Architecture

### 1. Backend (Go + Gin)
- Handles auth, course/assignment/submission APIs, and role checks
- Persists users, courses, enrollments, assignments, submissions, and evaluations
- Forwards evaluation requests to FastAPI

### 2. AI Service (FastAPI + OpenRouter)
- Evaluates text and PDF submissions
- Uses strict grading prompt rules
- Includes guardrails for irrelevant/unrelated answers

### 3. Frontend (React + Vite)
- Login/signup flows
- Course dashboard (create/join/list)
- Course detail page (create assignment, submit text/PDF, professor review)
- Student submissions list with AI evaluation display

## Current Features

- **Auth & JWT**
  - `POST /signup`
  - `POST /login`
  - `GET /profile`

- **Courses**
  - Professor creates courses with generated course code
  - Student joins via course code
  - Role-based course listing

- **Assignments**
  - Professor creates assignments with `title`, `question`, `answer_key`
  - Students only receive assignment `id`, `title`, `question`

- **Submissions & Evaluation**
  - `POST /submit` for text answers
  - `POST /submit-file` for PDF answers
  - AI evaluation stored in DB (`marks`, `feedback`, `confidence`)
  - `GET /my-submissions` for student submission history

- **Professor Review**
  - `GET /assignments/:id/submissions`
  - `PUT /evaluations/:id` to override marks/feedback

- **RAG Doubt Resolution (Phase 6)**
  - `POST /courses/:id/materials` to upload course material
  - `GET /courses/:id/materials` to list course materials
  - `POST /courses/:id/doubt` for context-grounded answers with citations

- **Adaptive Learning (Phase 7 - Initial)**
  - Student profiling per course from evaluation history
  - Proficiency levels: `beginner`, `intermediate`, `advanced`
  - Adaptive doubt responses based on proficiency + interaction history
  - `GET /courses/:id/student-profile` for current student profile

## Project Structure

```text
ai-teaching-assistant/
├── ai-service/
│   ├── main.py
│   └── requirements.txt
├── backend/
│   ├── main.go
│   ├── db.go
│   ├── models.go
│   ├── auth.go
│   ├── courses.go
│   ├── assignments.go
│   ├── submissions.go
│   ├── evaluations.go
│   ├── middleware.go
│   ├── jwt.go
│   └── .env.example
└── frontend/
    ├── src/
    └── package.json
```

## Setup

### 1) Run AI service

```bash
cd ai-service
pip install -r requirements.txt
uvicorn main:app --reload --port 8000
```

### 2) Run backend

```bash
cd backend
cp .env.example .env
go run .
```

### 3) Run frontend

```bash
cd frontend
npm install
npm run dev
```

## Environment Variables

### `ai-service/.env`

```bash
OPENROUTER_API_KEY=your_openrouter_api_key
```

### `backend/.env`

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=ai_teaching_assistant
JWT_SECRET=your_secret_key_here
```

## Backend API Snapshot

- `POST /signup`
- `POST /login`
- `GET /profile`
- `POST /courses`
- `POST /courses/join`
- `GET /courses`
- `POST /assignments`
- `GET /courses/:id/assignments`
- `POST /submit`
- `POST /submit-file`
- `GET /my-submissions`
- `GET /assignments/:id/submissions`
- `PUT /evaluations/:id`
- `GET /submissions/:id`
- `POST /courses/:id/materials`
- `GET /courses/:id/materials`
- `POST /courses/:id/doubt`
- `GET /courses/:id/student-profile`
- `GET /materials/:id/file`

## AI Service API Snapshot

- `POST /evaluate`
- `POST /evaluate-multiple`
- `POST /evaluate-file`
