'use client'

import { useState } from 'react'

export default function UploadPage() {
  const [file, setFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [message, setMessage] = useState('')

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setFile(e.target.files[0])
      setMessage('')
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!file) {
      setMessage('Please select a file first')
      return
    }

    setUploading(true)
    setMessage('Upload functionality will be connected to your API soon!')
    
    // This is where we'll connect to your backend API
    // For now, we're just showing that the interface works
    
    setTimeout(() => {
      setUploading(false)
      setMessage(`File "${file.name}" selected successfully. API connection coming next!`)
    }, 1000)
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-4" style={{ backgroundColor: '#1B2240' }}>
      <div className="max-w-md w-full rounded-xl shadow-2xl p-8" style={{ backgroundColor: '#1B2240', border: '1px solid rgba(57, 225, 157, 0.2)' }}>
        <h1 className="text-3xl font-bold mb-2" style={{ color: '#FFF9F0' }}>Upload Computing Job</h1>
        <p className="mb-6" style={{ color: '#6C7280' }}>Submit your workload to the distributed network</p>
        
        <form onSubmit={handleSubmit} className="space-y-6">
          {/* File Input Area */}
          <div 
            className="border-2 border-dashed rounded-lg p-6 text-center transition-all hover:shadow-lg"
            style={{ 
              borderColor: 'rgba(57, 225, 157, 0.3)',
              backgroundColor: 'rgba(57, 225, 157, 0.05)'
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.borderColor = 'rgba(57, 225, 157, 0.5)'
              e.currentTarget.style.backgroundColor = 'rgba(57, 225, 157, 0.1)'
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.borderColor = 'rgba(57, 225, 157, 0.3)'
              e.currentTarget.style.backgroundColor = 'rgba(57, 225, 157, 0.05)'
            }}
          >
            <input
              type="file"
              onChange={handleFileChange}
              className="hidden"
              id="file-input"
              accept=".js,.py,.wasm"
            />
            <label htmlFor="file-input" className="cursor-pointer">
              <div className="space-y-2">
                <div style={{ color: '#FFF9F0' }}>
                  Click to browse or drag and drop
                </div>
                <div className="text-sm" style={{ color: '#6C7280' }}>
                  JavaScript, Python, or WASM files
                </div>
              </div>
            </label>
            
            {file && (
              <div className="mt-4 text-sm" style={{ color: '#39E19D' }}>
                <span className="font-semibold">✓ {file.name}</span>
                <div className="text-xs mt-1" style={{ color: '#6C7280' }}>
                  Size: {(file.size / 1024).toFixed(2)} KB
                </div>
              </div>
            )}
          </div>

          {/* Submit Button */}
          <button
            type="submit"
            disabled={!file || uploading}
            className="w-full rounded-lg py-3 font-semibold transition-all transform hover:scale-105 disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none"
            style={{
              backgroundColor: file && !uploading ? '#39E19D' : '#6C7280',
              color: '#1B2240',
              boxShadow: file && !uploading ? '0 4px 20px rgba(57, 225, 157, 0.3)' : 'none'
            }}
            onMouseEnter={(e) => {
              if (file && !uploading) {
                e.currentTarget.style.backgroundColor = '#64F2C6'
              }
            }}
            onMouseLeave={(e) => {
              if (file && !uploading) {
                e.currentTarget.style.backgroundColor = '#39E19D'
              }
            }}
          >
            {uploading ? 'Processing...' : 'Submit Job'}
          </button>

          {/* Status Message */}
          {message && (
            <div className="text-center text-sm rounded-lg p-3" 
                 style={{ 
                   backgroundColor: 'rgba(57, 225, 157, 0.1)', 
                   color: '#FFF9F0',
                   border: '1px solid rgba(57, 225, 157, 0.3)'
                 }}>
              {message}
            </div>
          )}
        </form>
      </div>
    </div>
  )
}