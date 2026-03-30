import json
import re

import fitz  # PyMuPDF
import requests
from fastapi import FastAPI, File, Form, HTTPException, UploadFile
from pydantic import BaseModel

from dotenv import load_dotenv
import os

load_dotenv()
API_KEY = os.getenv("OPENROUTER_API_KEY")

app = FastAPI()


class EvaluateRequest(BaseModel):
    question: str
    student_answer: str
    reference_answer: str


class EvaluateResponse(BaseModel):
    marks: float
    feedback: str
    mistakes: list[str]
    suggestions: list[str]
    confidence: float


class EvaluateFileResponse(EvaluateResponse):
    extracted_text: str


class ParsedQuestion(BaseModel):
    question: str
    student_answer: str


class EvaluateMultipleRequest(BaseModel):
    questions: list[EvaluateRequest]


class QuestionWiseResult(BaseModel):
    question: str
    marks: float
    feedback: str
    mistakes: list[str]
    suggestions: list[str]
    confidence: float


class EvaluateMultipleResponse(BaseModel):
    results: list[QuestionWiseResult]
    total_marks: float
    overall_feedback: str
    common_mistakes: list[str]
    improvement_plan: list[str]


class EvaluateFileDetailedResponse(BaseModel):
    extracted_text: str
    questions: list[ParsedQuestion]
    results: list[QuestionWiseResult]
    total_marks: float
    overall_feedback: str
    common_mistakes: list[str]
    improvement_plan: list[str]


MAX_STUDENT_ANSWER_CHARS = 8000
QUESTION_STOPWORDS = {
    "a", "an", "the", "and", "or", "to", "of", "in", "on", "for", "with",
    "is", "are", "was", "were", "be", "being", "been", "as", "at", "by",
    "from", "that", "this", "these", "those", "it", "its", "into", "about",
    "explain", "describe", "discuss", "define", "what", "why", "how", "when",
    "where", "which", "who", "whom", "whose", "compare", "differentiate",
    "list", "state", "write",
}


def extract_question_keywords(text: str) -> list[str]:
    tokens = re.findall(r"[a-zA-Z0-9]+", text.lower())
    keywords = [token for token in tokens if len(token) >= 3 and token not in QUESTION_STOPWORDS]
    return list(dict.fromkeys(keywords))


def should_short_circuit_evaluation(question: str, student_answer: str) -> bool:
    answer = (student_answer or "").strip()
    if len(answer) > MAX_STUDENT_ANSWER_CHARS:
        return True

    question_keywords = extract_question_keywords(question or "")
    if not question_keywords:
        return False

    answer_tokens = set(re.findall(r"[a-zA-Z0-9]+", answer.lower()))
    has_overlap = any(keyword in answer_tokens for keyword in question_keywords)
    return not has_overlap


def run_evaluation(payload: EvaluateRequest) -> EvaluateResponse:
    import os
    import re
    import json as json_lib
    from dotenv import load_dotenv

    if should_short_circuit_evaluation(payload.question, payload.student_answer):
        return EvaluateResponse(
            marks=2,
            feedback="The submitted content does not answer the question and appears unrelated.",
            mistakes=["The response does not address the key concepts in the question."],
            suggestions=["Focus on the exact question keywords and answer the asked concept directly."],
            confidence=0.9,
        )

    load_dotenv()
    api_key = os.getenv("OPENROUTER_API_KEY")
    if not api_key:
        raise HTTPException(status_code=500, detail="OPENROUTER_API_KEY is not set")

    prompt = f"""
You are an academic evaluator grading a student's answer.

IMPORTANT:
- Return STRICT JSON only.
- Do not include markdown.
- Do not include code fences.
- Do not include any extra text before or after JSON.
- Output must be exactly one JSON object.

Grading rubric:
- 9-10: Almost perfect, matches key concepts
- 6-8: Partially correct, some missing ideas
- 3-5: Basic understanding but major gaps
- 0-2: Incorrect or very poor answer

Strict grading rule:
- If the student's answer is unrelated to the question or clearly incorrect, assign marks below 3.
- Do not give generous marks for irrelevant, generic, or off-topic responses.
- If the student_answer is unrelated, irrelevant, or does not address the question, assign very low marks (0-3) and explicitly state that the answer is irrelevant.
- DO NOT assume missing information. Only evaluate based on the provided student_answer.
- If the student_answer is unrelated, irrelevant, or does not address the question, assign very low marks (0-3) and explicitly state that the answer is irrelevant.

Evaluation scope rule:
- DO NOT assume missing information. Only evaluate based on the provided student_answer.

Evaluation instructions:
1) Identify specific mistakes in the student's answer.
2) Identify missing concepts compared to the reference answer.
3) Suggest concrete improvements the student can apply.
4) Provide detailed feedback in at least 4-5 sentences explaining strengths, weaknesses, and improvements.

Feedback quality rules:
- feedback must be at least 4-5 complete sentences.
- Explicitly mention:
  a) what is correct in the student's answer,
  b) what is missing compared to the reference answer,
  c) what is incorrect or misleading,
  d) specific suggestions to improve the answer.
- Keep feedback constructive, specific, and academic.
- Do not change the JSON schema; keep the same keys.

Mistake quality rules:
- Be specific and concept-level.
- Do NOT use vague phrases like "Incomplete definition" or "Needs more detail".
- Each mistake must name the exact missing or incorrect concept.
- Keep each mistake short and clear (one line each).

Example:
- Bad: "Incomplete definition"
- Good: "Did not mention simulation of human intelligence"

Evaluate:
Question: {payload.question}
Student Answer: {payload.student_answer}
Reference Answer: {payload.reference_answer}

Example output JSON:
{{
  "marks": 7.5,
  "feedback": "The answer correctly identifies one core concept and shows partial understanding of the topic. However, it misses important supporting points that are present in the reference answer. A few statements are oversimplified and one claim is technically inaccurate. To improve, include the missing mechanisms and define key terms more precisely. Add one concrete example to demonstrate complete understanding.",
  "mistakes": ["Confuses key term A with term B", "States an incomplete definition"],
  "suggestions": ["Define term A clearly in one sentence", "Add the missing concept about mechanism C"],
  "confidence": 0.84
}}
"""

    try:
        response = requests.post(
            "https://openrouter.ai/api/v1/chat/completions",
            headers={
                "Authorization": f"Bearer {api_key}",
                "Content-Type": "application/json",
                "HTTP-Referer": "http://localhost:8000",
                "X-Title": "AI Teaching Assistant",
            },
            json={
                "model": "mistralai/mistral-7b-instruct-v0.1",
                "messages": [
                    {"role": "user", "content": prompt},
                ],
                "temperature": 0,
            },
            timeout=30,
        )
        response.raise_for_status()
    except requests.HTTPError as exc:
        status_code = exc.response.status_code if exc.response is not None else 502
        error_text = exc.response.text if exc.response is not None else str(exc)
        print("OpenRouter HTTP error:", error_text)
        raise HTTPException(
            status_code=status_code,
            detail=f"OpenRouter error: {error_text}",
        ) from exc
    except requests.RequestException as exc:
        print("OpenRouter request error:", str(exc))
        raise HTTPException(
            status_code=502,
            detail=f"Could not connect to OpenRouter API: {str(exc)}",
        ) from exc

    try:
        result = response.json()
    except ValueError as exc:
        raise HTTPException(status_code=502, detail="OpenRouter returned invalid response") from exc

    print("OpenRouter full response:", result)

    try:
        output_text = result["choices"][0]["message"]["content"]
    except (KeyError, IndexError, TypeError) as exc:
        raise HTTPException(status_code=502, detail="OpenRouter response missing content") from exc

    print("Extracted assistant text:", output_text)

    # Handle messy output by extracting the first JSON object.
    json_match = re.search(r"\{[\s\S]*\}", output_text)
    candidate_json = json_match.group(0).strip() if json_match else output_text.strip()

    try:
        parsed = json_lib.loads(candidate_json)
        # Sometimes model returns JSON as a quoted JSON string. Unwrap once.
        if isinstance(parsed, str):
            parsed = json_lib.loads(parsed)

        mistakes = parsed.get("mistakes", [])
        suggestions = parsed.get("suggestions", [])
        if not isinstance(mistakes, list):
            mistakes = [str(mistakes)] if mistakes else []
        if not isinstance(suggestions, list):
            suggestions = [str(suggestions)] if suggestions else []

        return EvaluateResponse(
            marks=float(parsed["marks"]),
            feedback=str(parsed["feedback"]),
            mistakes=[str(item).strip() for item in mistakes if str(item).strip()],
            suggestions=[str(item).strip() for item in suggestions if str(item).strip()],
            confidence=float(parsed["confidence"]),
        )
    except (ValueError, KeyError, TypeError):
        fallback_text = output_text.strip()
        if not fallback_text:
            fallback_text = "Model response could not be parsed into JSON."
        return EvaluateResponse(
            marks=5,
            feedback=fallback_text,
            mistakes=[],
            suggestions=[],
            confidence=0.6,
        )


def run_overall_summary(results: list[QuestionWiseResult]) -> dict:
    import json as json_lib
    import os
    import re

    from dotenv import load_dotenv

    load_dotenv()
    api_key = os.getenv("OPENROUTER_API_KEY")
    if not api_key:
        return {
            "overall_feedback": "Summary unavailable because API key is missing.",
            "common_mistakes": [],
            "improvement_plan": [],
        }

    results_payload = [
        {
            "question": item.question,
            "marks": item.marks,
            "feedback": item.feedback,
            "mistakes": item.mistakes,
            "suggestions": item.suggestions,
        }
        for item in results
    ]

    prompt = f"""
You are an academic performance summarizer.

IMPORTANT:
- Return STRICT JSON only.
- Do not include markdown or code fences.
- Output must be exactly one JSON object.

Given these question-wise evaluation results:
{json_lib.dumps(results_payload, ensure_ascii=False)}

Generate:
{{
  "overall_feedback": "Short overall performance summary",
  "common_mistakes": ["Repeated mistake 1", "Repeated mistake 2"],
  "improvement_plan": ["Action step 1", "Action step 2"]
}}

Instructions:
- Summarize overall performance clearly and concisely.
- Identify repeated mistakes across multiple questions.
- Suggest actionable improvements the student can follow.
"""

    try:
        response = requests.post(
            "https://openrouter.ai/api/v1/chat/completions",
            headers={
                "Authorization": f"Bearer {api_key}",
                "Content-Type": "application/json",
                "HTTP-Referer": "http://localhost:8000",
                "X-Title": "AI Teaching Assistant",
            },
            json={
                "model": "mistralai/mistral-7b-instruct-v0.1",
                "messages": [{"role": "user", "content": prompt}],
                "temperature": 0,
            },
            timeout=30,
        )
        response.raise_for_status()
        result = response.json()
        output_text = result["choices"][0]["message"]["content"]
    except (requests.RequestException, ValueError, KeyError, IndexError, TypeError):
        return {
            "overall_feedback": "Overall performance is mixed. Review the feedback for each question and improve weak concepts.",
            "common_mistakes": [],
            "improvement_plan": [
                "Revise key concepts from the reference answers.",
                "Address repeated mistakes mentioned per question.",
            ],
        }

    json_match = re.search(r"\{[\s\S]*\}", output_text)
    candidate_json = json_match.group(0).strip() if json_match else output_text.strip()

    try:
        parsed = json_lib.loads(candidate_json)
        if isinstance(parsed, str):
            parsed = json_lib.loads(parsed)
        common_mistakes = parsed.get("common_mistakes", [])
        improvement_plan = parsed.get("improvement_plan", [])
        if not isinstance(common_mistakes, list):
            common_mistakes = [str(common_mistakes)] if common_mistakes else []
        if not isinstance(improvement_plan, list):
            improvement_plan = [str(improvement_plan)] if improvement_plan else []

        return {
            "overall_feedback": str(parsed.get("overall_feedback", "")).strip()
            or "Overall feedback not available.",
            "common_mistakes": [str(item).strip() for item in common_mistakes if str(item).strip()],
            "improvement_plan": [str(item).strip() for item in improvement_plan if str(item).strip()],
        }
    except (ValueError, KeyError, TypeError):
        return {
            "overall_feedback": "Overall performance is mixed. Review the feedback for each question and improve weak concepts.",
            "common_mistakes": [],
            "improvement_plan": [
                "Revise key concepts from the reference answers.",
                "Address repeated mistakes mentioned per question.",
            ],
        }


def extract_text_from_pdf(pdf_bytes: bytes) -> str:
    try:
        with fitz.open(stream=pdf_bytes, filetype="pdf") as pdf_document:
            text_parts = []
            for page in pdf_document:
                text_parts.append(page.get_text())
    except Exception as exc:
        raise HTTPException(status_code=400, detail="Unable to read PDF file") from exc

    extracted_text = "\n".join(text_parts).strip()
    if not extracted_text:
        raise HTTPException(status_code=400, detail="No text found in the uploaded PDF")

    return extracted_text


def split_into_question_answer_pairs(extracted_text: str) -> list[ParsedQuestion]:
    # Stop parsing before REFERENCES section.
    cleaned_text = re.split(r"(?i)\bREFERENCES\b", extracted_text, maxsplit=1)[0].strip()

    # Match question starts like 1.Why / 1. Why / 2.What
    split_pattern = re.compile(r"\n?\d+\.\s*")
    matches = list(split_pattern.finditer(cleaned_text))

    question_starters = (
        "what",
        "why",
        "how",
        "when",
        "where",
        "which",
        "who",
        "whom",
        "whose",
        "define",
        "explain",
        "describe",
        "discuss",
        "compare",
        "differentiate",
        "identify",
        "list",
        "state",
        "write",
    )

    questions: list[ParsedQuestion] = []
    for index, match in enumerate(matches):
        start = match.end()
        end = matches[index + 1].start() if index + 1 < len(matches) else len(cleaned_text)
        block = cleaned_text[start:end]
        block = block.strip()
        if not block:
            continue

        lines = [line.strip() for line in block.splitlines() if line.strip()]
        if not lines:
            continue

        # Build question from one or more lines.
        question_parts: list[str] = [re.sub(r"\s+", " ", lines[0]).strip()]
        line_index = 1
        while line_index < len(lines):
            question_so_far = " ".join(question_parts).strip()
            if question_so_far.endswith("?") or question_so_far.endswith("."):
                break

            current_line = re.sub(r"\s+", " ", lines[line_index]).strip()
            if not current_line:
                line_index += 1
                continue

            # Continue question if line looks like continuation.
            if current_line[0].islower() or len(current_line.split()) <= 6:
                question_parts.append(current_line)
                line_index += 1
                continue

            # Otherwise treat as start of answer paragraph.
            break

        question_line = " ".join(question_parts).strip()
        question_line = re.sub(r"\s+", " ", question_line).strip()
        answer_text = re.sub(r"\s+", " ", "\n".join(lines[line_index:])).strip()

        if not question_line or not answer_text:
            continue

        lower_question = question_line.lower()
        looks_like_question = (
            "?" in question_line
            or lower_question.startswith(question_starters)
        )
        if not looks_like_question:
            continue

        questions.append(
            ParsedQuestion(
                question=question_line,
                student_answer=answer_text,
            )
        )

    print("Questions detected:", len(questions))

    return questions


@app.post("/evaluate", response_model=EvaluateResponse)
def evaluate_answer(payload: EvaluateRequest) -> EvaluateResponse:
    return run_evaluation(payload)


@app.post("/evaluate-file", response_model=EvaluateFileDetailedResponse)
async def evaluate_file(
    file: UploadFile = File(...),
    reference_answer: str = Form(...),
    question: str = Form(""),
) -> EvaluateFileDetailedResponse:
    if not file.filename.lower().endswith(".pdf"):
        raise HTTPException(status_code=400, detail="Please upload a PDF file")
    if not reference_answer or not reference_answer.strip():
        raise HTTPException(status_code=400, detail="reference_answer is required")

    try:
        file_bytes = await file.read()
    except Exception as exc:
        raise HTTPException(status_code=400, detail="Could not read uploaded file") from exc

    extracted_text = extract_text_from_pdf(file_bytes)
    print("Extracted text length:", len(extracted_text))
    print("Extracted text:", extracted_text[:500])

    question = (question or "").strip()
    if question:
        print("Single-question mode enabled. Skipping split and evaluate-multiple.")
        single_result = run_evaluation(
            EvaluateRequest(
                question=question,
                student_answer=extracted_text,
                reference_answer=reference_answer,
            )
        )
        return EvaluateFileDetailedResponse(
            extracted_text=extracted_text,
            questions=[
                ParsedQuestion(
                    question=question,
                    student_answer=extracted_text,
                )
            ],
            results=[
                QuestionWiseResult(
                    question=question,
                    marks=single_result.marks,
                    feedback=single_result.feedback,
                    mistakes=single_result.mistakes,
                    suggestions=single_result.suggestions,
                    confidence=single_result.confidence,
                )
            ],
            total_marks=single_result.marks,
            overall_feedback=single_result.feedback,
            common_mistakes=single_result.mistakes,
            improvement_plan=single_result.suggestions,
        )

    parsed_questions = split_into_question_answer_pairs(extracted_text)
    if not parsed_questions:
        print("Question split failed. Using full extracted_text as student_answer.")
        single_result = run_evaluation(
            EvaluateRequest(
                question="Full submission response",
                student_answer=extracted_text,
                reference_answer=reference_answer,
            )
        )
        return EvaluateFileDetailedResponse(
            extracted_text=extracted_text,
            questions=[],
            results=[
                QuestionWiseResult(
                    question="Full submission response",
                    marks=single_result.marks,
                    feedback=single_result.feedback,
                    mistakes=single_result.mistakes,
                    suggestions=single_result.suggestions,
                    confidence=single_result.confidence,
                )
            ],
            total_marks=single_result.marks,
            overall_feedback=single_result.feedback,
            common_mistakes=single_result.mistakes,
            improvement_plan=single_result.suggestions,
        )

    evaluate_requests = [
        EvaluateRequest(
            question=item.question,
            student_answer=item.student_answer,
            reference_answer=reference_answer,
        )
        for item in parsed_questions
    ]

    multi_result = evaluate_multiple(EvaluateMultipleRequest(questions=evaluate_requests))

    return EvaluateFileDetailedResponse(
        extracted_text=extracted_text,
        questions=parsed_questions,
        results=multi_result.results,
        total_marks=multi_result.total_marks,
        overall_feedback=multi_result.overall_feedback,
        common_mistakes=multi_result.common_mistakes,
        improvement_plan=multi_result.improvement_plan,
    )


@app.post("/evaluate-multiple", response_model=EvaluateMultipleResponse)
def evaluate_multiple(payload: EvaluateMultipleRequest) -> EvaluateMultipleResponse:
    if not payload.questions:
        raise HTTPException(status_code=400, detail="questions list cannot be empty")

    results: list[QuestionWiseResult] = []
    total_marks = 0.0

    for index, question_payload in enumerate(payload.questions, start=1):
        try:
            evaluation = run_evaluation(question_payload)
        except HTTPException as exc:
            raise HTTPException(
                status_code=exc.status_code,
                detail=f"Failed to evaluate question {index}: {exc.detail}",
            ) from exc
        except Exception as exc:
            raise HTTPException(
                status_code=500,
                detail=f"Unexpected error while evaluating question {index}",
            ) from exc

        results.append(
            QuestionWiseResult(
                question=question_payload.question,
                marks=evaluation.marks,
                feedback=evaluation.feedback,
                mistakes=evaluation.mistakes,
                suggestions=evaluation.suggestions,
                confidence=evaluation.confidence,
            )
        )
        total_marks += evaluation.marks

    summary = run_overall_summary(results)

    return EvaluateMultipleResponse(
        results=results,
        total_marks=total_marks,
        overall_feedback=summary["overall_feedback"],
        common_mistakes=summary["common_mistakes"],
        improvement_plan=summary["improvement_plan"],
    )
