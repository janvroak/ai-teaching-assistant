import Header from '../components/Header'
import Layout from '../components/Layout'

function HomePage() {
  return (
    <Layout>
      <div>
        <Header />
        <p className="mt-3 text-slate-600">
          Frontend scaffold is ready with Vite, React, Tailwind CSS, and Axios.
        </p>
      </div>
    </Layout>
  )
}

export default HomePage
