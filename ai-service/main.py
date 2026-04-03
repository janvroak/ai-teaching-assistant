import json
import math
import threading
import re

import fitz  # PyMuPDF
import requests
from fastapi import FastAPI, File, Form, HTTPException, UploadFile
from pydantic import BaseModel, Field

from dotenv import load_dotenv
import os

load_dotenv()
API_KEY = os.getenv("OPENROUTER_API_KEY")

app = FastAPI()

_EMBEDDING_MODEL_LOCK = threading.Lock()
_EMBEDDING_MODEL = None
_EMBEDDING_MODEL_LOAD_FAILED = False


class EvaluateRequest(BaseModel):
    question: str
    student_answer: str
    reference_answer: str


class EvaluateResponse(BaseModel):
    marks: float
    score: float
    feedback: str
    correct_points: list[str]
    wrong_points: list[str]
    missing_concepts: list[str]
    strong_topics: list[str]
    weak_topics: list[str]
    mistakes: list[str]
    suggestions: list[str]
    confidence: float
    topics: list[str]


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
    score: float
    feedback: str
    correct_points: list[str]
    wrong_points: list[str]
    missing_concepts: list[str]
    strong_topics: list[str]
    weak_topics: list[str]
    mistakes: list[str]
    suggestions: list[str]
    confidence: float
    topics: list[str]


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


class ExtractPDFTextOnlyResponse(BaseModel):
    extracted_text: str


class AnswerWithContextRequest(BaseModel):
    question: str
    contexts: list[str]
    proficiency_level: str | None = None
    weak_topics: list[str] = Field(default_factory=list)
    recent_mistakes: list[str] = Field(default_factory=list)


class AnswerWithContextResponse(BaseModel):
    answer: str
    confidence: float
    citations: list[str] = []


def adaptation_instructions(proficiency_level: str) -> str:
    level = (proficiency_level or "").strip().lower()
    if level == "advanced":
        return (
            "- Keep the explanation concise and technically deep.\n"
            "- Use precise domain terminology when relevant.\n"
            "- Skip unnecessary basic background."
        )
    if level == "beginner":
        return (
            "- Explain step-by-step.\n"
            "- Use simple language and avoid jargon.\n"
            "- Add one easy example grounded in the provided context."
        )
    return (
        "- Use moderate detail with clear structure.\n"
        "- Include one practical example from context.\n"
        "- Use some technical terms, but keep it accessible."
    )


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


def clamp01(value: float) -> float:
    return max(0.0, min(1.0, value))


def cosine_similarity(a: list[float], b: list[float]) -> float:
    if not a or not b or len(a) != len(b):
        return 0.0

    dot = 0.0
    norm_a = 0.0
    norm_b = 0.0
    for x, y in zip(a, b):
        dot += x * y
        norm_a += x * x
        norm_b += y * y

    if norm_a <= 0 or norm_b <= 0:
        return 0.0
    return dot / (math.sqrt(norm_a) * math.sqrt(norm_b))


def normalize_similarity(raw_cosine: float) -> float:
    # cosine in [-1, 1] -> normalized similarity in [0, 1]
    return clamp01((raw_cosine + 1.0) / 2.0)


def token_jaccard_similarity(a_text: str, b_text: str) -> float:
    a_tokens = set(re.findall(r"[a-zA-Z0-9]+", (a_text or "").lower()))
    b_tokens = set(re.findall(r"[a-zA-Z0-9]+", (b_text or "").lower()))
    if not a_tokens or not b_tokens:
        return 0.0
    intersection = len(a_tokens.intersection(b_tokens))
    union = len(a_tokens.union(b_tokens))
    if union == 0:
        return 0.0
    return clamp01(intersection / union)


def embed_with_sentence_transformers(texts: list[str]) -> list[list[float]] | None:
    global _EMBEDDING_MODEL
    global _EMBEDDING_MODEL_LOAD_FAILED

    if _EMBEDDING_MODEL_LOAD_FAILED:
        return None

    with _EMBEDDING_MODEL_LOCK:
        if _EMBEDDING_MODEL is None:
            try:
                from sentence_transformers import SentenceTransformer
                model_name = os.getenv("SENTENCE_TRANSFORMER_MODEL", "all-MiniLM-L6-v2")
                _EMBEDDING_MODEL = SentenceTransformer(model_name)
            except Exception as exc:
                print("SentenceTransformer unavailable:", str(exc))
                _EMBEDDING_MODEL_LOAD_FAILED = True
                return None

    try:
        vectors = _EMBEDDING_MODEL.encode(texts, convert_to_numpy=True, normalize_embeddings=True)
        return [vector.astype(float).tolist() for vector in vectors]
    except Exception as exc:
        print("SentenceTransformer embedding failed:", str(exc))
        return None


def embed_with_openai(texts: list[str]) -> list[list[float]] | None:
    api_key = os.getenv("OPENAI_API_KEY", "").strip()
    if not api_key:
        return None

    model = os.getenv("OPENAI_EMBEDDING_MODEL", "text-embedding-3-small").strip() or "text-embedding-3-small"
    try:
        response = requests.post(
            "https://api.openai.com/v1/embeddings",
            headers={
                "Authorization": f"Bearer {api_key}",
                "Content-Type": "application/json",
            },
            json={
                "model": model,
                "input": texts,
            },
            timeout=30,
        )
        response.raise_for_status()
        payload = response.json()
        data = payload.get("data", [])
        if not isinstance(data, list) or len(data) != len(texts):
            return None
        embeddings: list[list[float]] = []
        for item in data:
            vector = item.get("embedding", [])
            if not isinstance(vector, list) or not vector:
                return None
            embeddings.append([float(v) for v in vector])
        return embeddings
    except Exception as exc:
        print("OpenAI embedding fallback failed:", str(exc))
        return None


def semantic_similarity(student_answer: str, reference_answer: str) -> float:
    student = (student_answer or "").strip()
    reference = (reference_answer or "").strip()
    if not student or not reference:
        return 0.0

    vectors = embed_with_sentence_transformers([student, reference])
    if vectors and len(vectors) == 2:
        raw = cosine_similarity(vectors[0], vectors[1])
        return normalize_similarity(raw)

    vectors = embed_with_openai([student, reference])
    if vectors and len(vectors) == 2:
        raw = cosine_similarity(vectors[0], vectors[1])
        return normalize_similarity(raw)

    # Final fallback if embedding backends are unavailable.
    return token_jaccard_similarity(student, reference)


def apply_similarity_penalty(
    marks: float,
    feedback: str,
    mistakes: list[str],
    suggestions: list[str],
    similarity: float,
    allow_not_relevant_feedback: bool = True,
) -> tuple[float, str, list[str], list[str]]:
    if similarity >= 0.3:
        return marks, feedback, mistakes, suggestions

    penalized_marks = max(0.0, min(marks, marks * 0.6))
    penalty_line = "Answer not relevant."

    updated_feedback = (feedback or "").strip()
    if allow_not_relevant_feedback:
        if penalty_line.lower() not in updated_feedback.lower():
            if updated_feedback:
                updated_feedback = f"{updated_feedback} {penalty_line}"
            else:
                updated_feedback = penalty_line

    updated_mistakes = list(mistakes or [])
    if allow_not_relevant_feedback and all("not relevant" not in str(item).lower() for item in updated_mistakes):
        updated_mistakes.append("Answer not relevant to the expected reference concepts.")

    updated_suggestions = list(suggestions or [])
    if all("reference answer" not in str(item).lower() for item in updated_suggestions):
        updated_suggestions.append("Align your response with the reference answer's key concepts.")

    return penalized_marks, updated_feedback, updated_mistakes, updated_suggestions


def clean_text(value: str) -> str:
    text = str(value or "")
    text = text.replace("\r", " ").replace("\n", " ")
    text = re.sub(r"\s+", " ", text).strip()
    return text


def normalize_concept_list(items: list[str], limit: int = 20) -> list[str]:
    seen: set[str] = set()
    result: list[str] = []
    for item in items or []:
        cleaned = clean_text(item)
        if not cleaned:
            continue
        lowered = cleaned.lower()
        if lowered in seen:
            continue
        seen.add(lowered)
        result.append(cleaned)
        if limit > 0 and len(result) >= limit:
            break
    return result


def derive_topics_from_points(
    correct_points: list[str],
    wrong_points: list[str],
    missing_concepts: list[str],
) -> tuple[list[str], list[str], list[str]]:
    # Strict rule: topics are derived only from concept-level evaluation points.
    strong_topics = normalize_concept_list(correct_points, 12)
    weak_topics = normalize_concept_list((wrong_points or []) + (missing_concepts or []), 12)
    all_topics = normalize_concept_list(strong_topics + weak_topics, 20)
    return strong_topics, weak_topics, all_topics


def clamp_score(value: float) -> float:
    return max(0.0, min(10.0, float(value)))


def enforce_structured_score(
    raw_score: float,
    correct_points: list[str],
    wrong_points: list[str],
    missing_concepts: list[str],
    student_answer: str,
    reference_answer: str,
) -> float:
    score = clamp_score(raw_score)
    total_concepts = len(correct_points) + len(wrong_points) + len(missing_concepts)
    if total_concepts <= 0:
        if not (student_answer or "").strip():
            return 0.0
        return score

    coverage = len(correct_points) / float(total_concepts)
    deterministic_score = (coverage * 10.0) - (len(wrong_points) * 1.2) - (len(missing_concepts) * 0.8)
    deterministic_score = clamp_score(deterministic_score)
    score = clamp_score((0.7 * score) + (0.3 * deterministic_score))

    if len(correct_points) == 0 and (len(wrong_points) > 0 or len(missing_concepts) > 0):
        score = min(score, 2.9)
    elif coverage >= 0.7 and len(correct_points) >= 2:
        score = max(score, 7.0)
    elif 0 < coverage < 0.7:
        score = min(max(score, 4.0), 7.0)

    if token_jaccard_similarity(student_answer, reference_answer) < 0.08 and len(correct_points) == 0:
        score = min(score, 2.0)

    return clamp_score(score)


def run_evaluation(payload: EvaluateRequest) -> EvaluateResponse:
    import json as json_lib
    from dotenv import load_dotenv

    question = clean_text(payload.question)
    student_answer = clean_text(payload.student_answer)
    reference_answer = clean_text(payload.reference_answer)
    print("QUESTION:", question)
    print("REFERENCE:", reference_answer[:300])
    print("STUDENT:", student_answer[:300])
    similarity = semantic_similarity(student_answer, reference_answer)

    if len(reference_answer) < 10:
        strong_topics, weak_topics, topics = derive_topics_from_points([], [], [])
        return EvaluateResponse(
            marks=5,
            score=5,
            feedback="Reference answer missing, partial evaluation only",
            correct_points=[],
            wrong_points=[],
            missing_concepts=[],
            strong_topics=strong_topics,
            weak_topics=weak_topics,
            mistakes=[],
            suggestions=["Add a detailed reference answer for full grading accuracy."],
            confidence=0.5,
            topics=topics,
        )

    if should_short_circuit_evaluation(question, student_answer):
        llm_confidence = 0.3
        final_confidence = clamp01((0.6 * llm_confidence) + (0.4 * similarity))
        correct_points: list[str] = []
        wrong_points = ["The answer does not address the key concepts from the answer key."]
        missing_concepts = ["Key concepts from the answer key were not covered."]
        strong_topics, weak_topics, topics = derive_topics_from_points(correct_points, wrong_points, missing_concepts)
        mistakes = normalize_concept_list(wrong_points + missing_concepts, 20)
        suggestions = ["Focus on the core concepts required by the question and answer key."]
        marks, feedback, mistakes, suggestions = apply_similarity_penalty(
            marks=2,
            feedback="The submitted content does not answer the question and appears unrelated.",
            mistakes=mistakes,
            suggestions=suggestions,
            similarity=similarity,
        )
        return EvaluateResponse(
            marks=marks,
            score=marks,
            feedback=feedback,
            correct_points=correct_points,
            wrong_points=wrong_points,
            missing_concepts=missing_concepts,
            strong_topics=strong_topics,
            weak_topics=weak_topics,
            mistakes=mistakes,
            suggestions=suggestions,
            confidence=final_confidence,
            topics=topics,
        )

    load_dotenv()
    api_key = os.getenv("OPENROUTER_API_KEY")
    if not api_key:
        raise HTTPException(status_code=500, detail="OPENROUTER_API_KEY is not set")

    system_prompt = """
You are a strict academic evaluator.

You MUST:
- Compare student answer with answer key
- Identify concepts (not sentences)
- Penalize incorrect statements heavily
- Reward correctness proportionally
- NEVER give random marks
- Avoid vague statements and generic feedback
""".strip()

    user_prompt = f"""
Question:
{question}

Answer Key:
{reference_answer}

Student Answer:
{student_answer}

---

TASK:

1. Extract key concepts from answer key
2. For each concept:
   - Check if student covered it correctly, incorrectly, or missed it

3. Output STRICT JSON:
{{
  "score": 0,
  "correct_points": [],
  "wrong_points": [],
  "missing_concepts": [],
  "strong_topics": [],
  "weak_topics": [],
  "feedback": "detailed paragraph explaining performance",
  "confidence": 0.0,
  "mistakes": []
}}

RULES:
- If student includes wrong statements -> reduce score significantly
- If answer is partially correct -> mid score (4-7)
- If mostly correct -> high score (7-10)
- If irrelevant -> score < 3
- Concepts must be short technical phrases, not full sentences
- Return JSON only, no markdown, no code fences, no extra text.
""".strip()

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
                    {"role": "system", "content": system_prompt},
                    {"role": "user", "content": user_prompt},
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

        def ensure_list(value: object) -> list[str]:
            if isinstance(value, list):
                return [str(item) for item in value]
            if value is None:
                return []
            text_value = clean_text(str(value))
            return [text_value] if text_value else []

        correct_points = normalize_concept_list(ensure_list(parsed.get("correct_points", [])), 20)
        wrong_points = normalize_concept_list(ensure_list(parsed.get("wrong_points", [])), 20)
        missing_concepts = normalize_concept_list(ensure_list(parsed.get("missing_concepts", [])), 20)
        strong_topics, weak_topics, topics = derive_topics_from_points(correct_points, wrong_points, missing_concepts)

        feedback_text = clean_text(str(parsed.get("feedback", "")).strip())
        if not feedback_text:
            parts: list[str] = []
            if correct_points:
                parts.append("Correct concepts: " + "; ".join(correct_points))
            if wrong_points:
                parts.append("Incorrect concepts: " + "; ".join(wrong_points))
            if missing_concepts:
                parts.append("Missing concepts: " + "; ".join(missing_concepts))
            feedback_text = " ".join(parts).strip() or "Concept-level evaluation completed."

        derived_suggestions: list[str] = normalize_concept_list(missing_concepts[:4] + wrong_points[:3], 8)

        parsed_mistakes = normalize_concept_list(ensure_list(parsed.get("mistakes", [])), 20)
        cleaned_mistakes = normalize_concept_list(parsed_mistakes + wrong_points + missing_concepts, 25)
        cleaned_suggestions = derived_suggestions

        try:
            llm_confidence = clamp01(float(parsed.get("confidence", 0.6)))
        except (ValueError, TypeError):
            llm_confidence = 0.6
        final_confidence = clamp01((0.6 * llm_confidence) + (0.4 * similarity))

        try:
            marks = float(parsed.get("score", parsed.get("marks", 0)))
        except (ValueError, TypeError):
            marks = 0.0
        marks = enforce_structured_score(
            raw_score=marks,
            correct_points=correct_points,
            wrong_points=wrong_points,
            missing_concepts=missing_concepts,
            student_answer=student_answer,
            reference_answer=reference_answer,
        )

        feedback = feedback_text
        has_any_correct_points = len(correct_points) > 0
        marks, feedback, cleaned_mistakes, cleaned_suggestions = apply_similarity_penalty(
            marks=marks,
            feedback=feedback,
            mistakes=cleaned_mistakes,
            suggestions=cleaned_suggestions,
            similarity=similarity,
            allow_not_relevant_feedback=not has_any_correct_points,
        )
        if has_any_correct_points:
            feedback = re.sub(r"(?i)\banswer not relevant\.?\b", "", feedback).strip()
            feedback = re.sub(r"\s{2,}", " ", feedback)
            cleaned_mistakes = [
                item for item in cleaned_mistakes
                if "not relevant" not in item.lower()
            ]

        return EvaluateResponse(
            marks=marks,
            score=marks,
            feedback=feedback,
            correct_points=correct_points,
            wrong_points=wrong_points,
            missing_concepts=missing_concepts,
            strong_topics=strong_topics,
            weak_topics=weak_topics,
            mistakes=cleaned_mistakes,
            suggestions=cleaned_suggestions,
            confidence=final_confidence,
            topics=topics,
        )
    except (ValueError, KeyError, TypeError):
        fallback_text = output_text.strip()
        if not fallback_text:
            fallback_text = "Model response could not be parsed into JSON."
        llm_confidence = 0.6
        final_confidence = clamp01((0.6 * llm_confidence) + (0.4 * similarity))
        correct_points: list[str] = []
        wrong_points = ["The evaluator could not parse concept-level output."]
        missing_concepts: list[str] = []
        strong_topics, weak_topics, topics = derive_topics_from_points(correct_points, wrong_points, missing_concepts)
        mistakes = normalize_concept_list(wrong_points, 10)
        suggestions = ["Resubmit with clearer concept-focused content for reliable grading."]
        marks, feedback, mistakes, suggestions = apply_similarity_penalty(
            marks=5,
            feedback=fallback_text,
            mistakes=mistakes,
            suggestions=suggestions,
            similarity=similarity,
        )
        return EvaluateResponse(
            marks=marks,
            score=marks,
            feedback=feedback,
            correct_points=correct_points,
            wrong_points=wrong_points,
            missing_concepts=missing_concepts,
            strong_topics=strong_topics,
            weak_topics=weak_topics,
            mistakes=mistakes,
            suggestions=suggestions,
            confidence=final_confidence,
            topics=topics,
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


@app.post("/extract-pdf-text", response_model=ExtractPDFTextOnlyResponse)
async def extract_pdf_text(file: UploadFile = File(...)) -> ExtractPDFTextOnlyResponse:
    if not file.filename.lower().endswith(".pdf"):
        raise HTTPException(status_code=400, detail="Please upload a PDF file")

    try:
        file_bytes = await file.read()
    except Exception as exc:
        raise HTTPException(status_code=400, detail="Could not read uploaded file") from exc

    extracted_text = extract_text_from_pdf(file_bytes)
    return ExtractPDFTextOnlyResponse(extracted_text=extracted_text)


@app.post("/answer-with-context", response_model=AnswerWithContextResponse)
def answer_with_context(payload: AnswerWithContextRequest) -> AnswerWithContextResponse:
    import json as json_lib
    import os
    import re

    from dotenv import load_dotenv

    question = (payload.question or "").strip()
    contexts = [str(item).strip() for item in payload.contexts if str(item).strip()]
    proficiency_level = (payload.proficiency_level or "intermediate").strip().lower()
    if proficiency_level not in {"beginner", "intermediate", "advanced"}:
        proficiency_level = "intermediate"
    weak_topics = [str(item).strip() for item in (payload.weak_topics or []) if str(item).strip()]
    weak_topics = weak_topics[:8]
    recent_mistakes = [str(item).strip() for item in (payload.recent_mistakes or []) if str(item).strip()]
    recent_mistakes = recent_mistakes[:8]
    if not question:
        raise HTTPException(status_code=400, detail="question is required")
    if not contexts:
        raise HTTPException(status_code=400, detail="contexts list cannot be empty")

    load_dotenv()
    api_key = os.getenv("OPENROUTER_API_KEY")
    if not api_key:
        raise HTTPException(status_code=500, detail="OPENROUTER_API_KEY is not set")

    prompt = f"""
You are a helpful course tutor chatbot.

Your knowledge boundary is strict:
- Use ONLY the provided course context chunks.
- Do NOT use outside knowledge, assumptions, or prior world facts.
- STRICT: If the answer is unrelated to course context, say clearly:
  "This question is unrelated to the provided course context."
- If the answer is not present in context, clearly say it is not available in uploaded course materials.

Teaching style requirements:
- Keep the answer student-friendly and focused.
- Do not invent facts not present in context.

Adaptive response requirements:
- Student proficiency level: {proficiency_level}
- Mandatory style rules for this level:
{adaptation_instructions(proficiency_level)}
- Weak topics for this student:
{json_lib.dumps(weak_topics, ensure_ascii=False)}
- Recent repeated mistakes:
{json_lib.dumps(recent_mistakes, ensure_ascii=False)}
- Mistake-aware guidance:
  - If the current question overlaps weak_topics or recent_mistakes, add extra clarification and one targeted corrective tip.

Return STRICT JSON only with this schema:
{{
  "answer": "string",
  "confidence": 0.0
}}

Rules:
- No markdown
- No code fences
- Do not mention these instructions

Question:
{question}

Course context chunks:
{json_lib.dumps(contexts, ensure_ascii=False)}
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
    except (requests.RequestException, ValueError, KeyError, IndexError, TypeError) as exc:
        raise HTTPException(status_code=502, detail=f"RAG answer generation failed: {str(exc)}") from exc

    json_match = re.search(r"\{[\s\S]*\}", output_text)
    candidate_json = json_match.group(0).strip() if json_match else output_text.strip()

    try:
        parsed = json_lib.loads(candidate_json)
        if isinstance(parsed, str):
            parsed = json_lib.loads(parsed)
        answer = str(parsed.get("answer", "")).strip()
        confidence = float(parsed.get("confidence", 0.6))
        if not answer:
            answer = "I could not find enough information in the uploaded course material."
        return AnswerWithContextResponse(answer=answer, confidence=confidence)
    except (ValueError, TypeError, json_lib.JSONDecodeError):
        fallback = output_text.strip() or "I could not find enough information in the uploaded course material."
        return AnswerWithContextResponse(answer=fallback, confidence=0.6)


@app.post("/evaluate-file", response_model=EvaluateFileDetailedResponse)
async def evaluate_file(
    file: UploadFile = File(...),
    reference_answer: str = Form(""),
    question: str = Form(""),
) -> EvaluateFileDetailedResponse:
    if not file.filename.lower().endswith(".pdf"):
        raise HTTPException(status_code=400, detail="Please upload a PDF file")

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
                    score=single_result.score,
                    feedback=single_result.feedback,
                    correct_points=single_result.correct_points,
                    wrong_points=single_result.wrong_points,
                    missing_concepts=single_result.missing_concepts,
                    strong_topics=single_result.strong_topics,
                    weak_topics=single_result.weak_topics,
                    mistakes=single_result.mistakes,
                    suggestions=single_result.suggestions,
                    confidence=single_result.confidence,
                    topics=single_result.topics,
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
                    score=single_result.score,
                    feedback=single_result.feedback,
                    correct_points=single_result.correct_points,
                    wrong_points=single_result.wrong_points,
                    missing_concepts=single_result.missing_concepts,
                    strong_topics=single_result.strong_topics,
                    weak_topics=single_result.weak_topics,
                    mistakes=single_result.mistakes,
                    suggestions=single_result.suggestions,
                    confidence=single_result.confidence,
                    topics=single_result.topics,
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
                score=evaluation.score,
                feedback=evaluation.feedback,
                correct_points=evaluation.correct_points,
                wrong_points=evaluation.wrong_points,
                missing_concepts=evaluation.missing_concepts,
                strong_topics=evaluation.strong_topics,
                weak_topics=evaluation.weak_topics,
                mistakes=evaluation.mistakes,
                suggestions=evaluation.suggestions,
                confidence=evaluation.confidence,
                topics=evaluation.topics,
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
