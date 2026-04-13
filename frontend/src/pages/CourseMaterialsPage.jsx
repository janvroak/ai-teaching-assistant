import { useEffect, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import CourseShell from '../components/CourseShell'
import api from '../services/api'

function CourseMaterialsPage() {
  const [materials, setMaterials] = useState([])
  const [loading, setLoading] = useState(true)
  const [listError, setListError] = useState('')
  const [uploading, setUploading] = useState(false)
  const [uploadError, setUploadError] = useState('')
  const [uploadMessage, setUploadMessage] = useState('')
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [file, setFile] = useState(null)
  const [downloadingId, setDownloadingId] = useState(null)
  const [viewerError, setViewerError] = useState('')

  const { courseId } = useParams()

  const user = useMemo(() => {
    try {
      const raw = localStorage.getItem('user')
      return raw ? JSON.parse(raw) : null
    } catch {
      return null
    }
  }, [])

  const isProfessor = user?.role === 'professor'

  const loadMaterials = async () => {
    setLoading(true)
    setListError('')
    try {
      const response = await api.get(`/courses/${courseId}/materials`)
      setMaterials(Array.isArray(response.data) ? response.data : [])
    } catch (err) {
      setListError(err?.response?.data?.error || 'Failed to fetch course materials')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadMaterials()
  }, [courseId])

  const handleUpload = async (event) => {
    event.preventDefault()
    setUploadError('')
    setUploadMessage('')

    const trimmedTitle = title.trim()
    const trimmedContent = content.trim()

    if (!trimmedTitle) {
      setUploadError('Material title is required')
      return
    }
    if (!file && !trimmedContent) {
      setUploadError('Upload a PDF file or provide text content')
      return
    }
    if (file && trimmedContent) {
      setUploadError('Provide either PDF file or text content, not both')
      return
    }

    const formData = new FormData()
    formData.append('title', trimmedTitle)
    if (file) {
      formData.append('file', file)
    } else {
      formData.append('content', trimmedContent)
    }

    setUploading(true)
    try {
      await api.post(`/courses/${courseId}/materials`, formData)
      setUploadMessage('Material uploaded successfully. RAG can now use this content.')
      setTitle('')
      setContent('')
      setFile(null)
      await loadMaterials()
    } catch (err) {
      setUploadError(err?.response?.data?.error || 'Failed to upload material')
    } finally {
      setUploading(false)
    }
  }

  const openMaterialFile = async (material) => {
    if (!material?.has_file || !material?.file_url) return
    setViewerError('')
    setDownloadingId(material.id)
    try {
      const response = await api.get(material.file_url, { responseType: 'blob' })
      const fileUrl = URL.createObjectURL(response.data)
      window.open(fileUrl, '_blank', 'noopener,noreferrer')
      setTimeout(() => {
        URL.revokeObjectURL(fileUrl)
      }, 60_000)
    } catch (err) {
      setViewerError(err?.response?.data?.error || 'Failed to open material file')
    } finally {
      setDownloadingId(null)
    }
  }

  return (
    <CourseShell
      courseId={courseId}
      activeTab="materials"
      title="Course Materials"
      description="Upload and review learning materials that power course RAG answers."
    >
      <section className="space-y-6">
        {isProfessor ? (
          <form onSubmit={handleUpload} className="app-card space-y-4 p-6">
            <div>
              <h2 className="section-title">Upload Material</h2>
              <p className="section-subtitle">
                Add a PDF or plain text notes. Students and chatbot answers rely on this material.
              </p>
            </div>

            <div className="space-y-3">
              <input
                type="text"
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                className="input-field"
                placeholder="Material title"
              />
              <div className="grid gap-3 md:grid-cols-2">
                <label className="block rounded-xl border border-white/10 bg-[#0b1020] p-3 text-sm text-slate-300">
                  <span className="mb-2 block text-xs uppercase tracking-wide text-slate-400">PDF File</span>
                  <input
                    type="file"
                    accept=".pdf,application/pdf"
                    onChange={(event) => setFile(event.target.files?.[0] || null)}
                    className="block w-full text-xs text-slate-300 file:mr-3 file:rounded-lg file:border-0 file:bg-blue-500/20 file:px-3 file:py-2 file:text-xs file:font-medium file:text-blue-100 hover:file:bg-blue-500/30"
                  />
                </label>
                <textarea
                  value={content}
                  onChange={(event) => setContent(event.target.value)}
                  rows={5}
                  className="input-field resize-y"
                  placeholder="Or paste text content here"
                />
              </div>
              <button type="submit" disabled={uploading} className="btn-primary w-auto px-5 py-3">
                {uploading ? 'Uploading...' : 'Upload Material'}
              </button>
            </div>
            {uploadError ? <p className="text-sm text-red-300">{uploadError}</p> : null}
            {uploadMessage ? <p className="text-sm text-emerald-300">{uploadMessage}</p> : null}
          </form>
        ) : null}

        <div className="app-card p-6">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div>
              <h2 className="section-title">Uploaded Materials</h2>
              <p className="section-subtitle">This is the source pool used for doubt-resolution RAG.</p>
            </div>
            <button
              type="button"
              onClick={loadMaterials}
              className="rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-xs text-slate-300 transition-all duration-300 hover:border-blue-500/30 hover:bg-white/10"
            >
              Refresh
            </button>
          </div>

          <div className="mt-4 space-y-3">
            {loading ? <p className="text-sm text-slate-400">Loading materials...</p> : null}
            {listError ? <p className="text-sm text-red-300">{listError}</p> : null}
            {viewerError ? <p className="text-sm text-red-300">{viewerError}</p> : null}

            {!loading && !listError && materials.length === 0 ? (
              <p className="text-sm text-slate-300">No materials uploaded for this course yet.</p>
            ) : null}

            {!loading && !listError && materials.length > 0 ? (
              <div className="space-y-2">
                {materials.map((material) => (
                  <div key={material.id} className="rounded-xl border border-white/10 bg-[#0b1020] p-4">
                    <div className="flex flex-wrap items-center justify-between gap-3">
                      <div>
                        <p className="text-sm font-medium text-slate-100">{material.title}</p>
                        <p className="mt-1 text-xs text-slate-400">
                          Uploaded by {material.uploaded_by || 'Professor'} on{' '}
                          {material.created_at ? new Date(material.created_at).toLocaleString() : '-'}
                        </p>
                      </div>
                      {material.has_file ? (
                        <button
                          type="button"
                          onClick={() => openMaterialFile(material)}
                          disabled={downloadingId === material.id}
                          className="rounded-lg border border-blue-500/30 bg-blue-500/10 px-3 py-2 text-xs text-blue-200 transition-all duration-300 hover:bg-blue-500/20 disabled:opacity-60"
                        >
                          {downloadingId === material.id ? 'Opening...' : 'Open PDF'}
                        </button>
                      ) : (
                        <span className="status-chip border-emerald-500/30 bg-emerald-500/10 text-emerald-200">
                          <span className="status-dot bg-emerald-300" />
                          Text Material
                        </span>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            ) : null}
          </div>
        </div>
      </section>
    </CourseShell>
  )
}

export default CourseMaterialsPage
