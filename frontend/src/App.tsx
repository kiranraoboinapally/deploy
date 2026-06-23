import { useEffect, useState } from "react";

interface Product {
  id: string;
  squarespace_id: string;
  name: string;
  price: number;
  currency: string;
  pdf_filename: string;
}

// Dynamic script loader utility for Razorpay SDK
const loadRazorpayScript = (): Promise<boolean> => {
  return new Promise((resolve) => {
    if ((window as any).Razorpay) {
      resolve(true);
      return;
    }
    const script = document.createElement("script");
    script.src = "https://checkout.razorpay.com/v1/checkout.js";
    script.onload = () => resolve(true);
    script.onerror = () => resolve(false);
    document.body.appendChild(script);
  });
};

function App() {
  const [products, setProducts] = useState<Product[]>([]);
  const [selectedProduct, setSelectedProduct] = useState<Product | null>(null);
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [loading, setLoading] = useState(false);
  const [initialLoading, setInitialLoading] = useState(true);
  const [error, setError] = useState("");
  const [successToken, setSuccessToken] = useState("");

  // Fetch products on mount
  useEffect(() => {
    fetch("/api/products")
      .then((res) => {
        if (!res.ok) throw new Error("Failed to load products");
        return res.json();
      })
      .then((data) => {
        setProducts(data);
        if (data.length > 0) {
          setSelectedProduct(data[0]); // Select first product by default
        }
        setInitialLoading(false);
      })
      .catch((err) => {
        console.error(err);
        setError("Could not load products. Please check if backend is running.");
        setInitialLoading(false);
      });
  }, []);

  const handleCheckout = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedProduct) return;
    if (!email || !name) {
      setError("Please fill in your name and email.");
      return;
    }

    setLoading(true);
    setError("");

    try {
      // 1. Load Razorpay JS SDK
      const scriptLoaded = await loadRazorpayScript();
      if (!scriptLoaded) {
        throw new Error("Razorpay SDK failed to load. Are you offline?");
      }

      // 2. Call backend to initiate Razorpay order
      const res = await fetch("/api/checkout/create-order", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          product_id: selectedProduct.id,
          email,
          name,
        }),
      });

      if (!res.ok) {
        const errorText = await res.text();
        throw new Error(errorText || "Failed to create order on server");
      }

      const orderData = await res.json();

      // 3. Open Razorpay Checkout modal
      const options = {
        key: orderData.razorpay_key_id,
        amount: orderData.amount,
        currency: orderData.currency,
        name: "UniCommerce",
        description: orderData.product_name,
        order_id: orderData.razorpay_order_id,
        handler: async (response: any) => {
          setLoading(true);
          try {
            // 4. Verify payment signature on backend
            const verifyRes = await fetch("/api/checkout/verify-payment", {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({
                razorpay_order_id: response.razorpay_order_id,
                razorpay_payment_id: response.razorpay_payment_id,
                razorpay_signature: response.razorpay_signature,
              }),
            });

            if (!verifyRes.ok) {
              const verifyError = await verifyRes.text();
              throw new Error(verifyError || "Payment verification failed");
            }

            const verifyData = await verifyRes.json();
            setSuccessToken(verifyData.token);
          } catch (err: any) {
            setError(err.message || "Payment verification failed");
          } finally {
            setLoading(false);
          }
        },
        prefill: {
          name: name,
          email: email,
        },
        theme: {
          color: "#8b5cf6",
        },
        modal: {
          ondismiss: () => {
            setLoading(false);
          },
        },
      };

      const rzp = new (window as any).Razorpay(options);
      rzp.open();
    } catch (err: any) {
      setError(err.message || "Something went wrong during checkout.");
      setLoading(false);
    }
  };

  if (initialLoading) {
    return (
      <div className="container loading-container">
        <div className="loader"></div>
        <p>Loading digital products store...</p>
      </div>
    );
  }

  if (successToken) {
    return (
      <div className="container success-card">
        <div className="success-icon">✓</div>
        <h2 className="success-title">Payment Successful!</h2>
        <p className="success-desc">
          Your payment was processed successfully. We've sent an email to{" "}
          <strong>{email}</strong> containing your download link.
        </p>
        <div className="download-box">
          <p style={{ fontSize: "14px", color: "var(--text-secondary)" }}>
            Or download your PDF document directly now:
          </p>
          <a
            href={`/api/download/${successToken}`}
            className="btn-download"
            target="_blank"
            rel="noopener noreferrer"
          >
            Download PDF File
          </a>
          <p style={{ fontSize: "12px", color: "var(--text-secondary)", marginTop: "4px" }}>
            The link remains active for 24 hours.
          </p>
        </div>
        <button
          className="btn-primary"
          style={{ background: "rgba(255,255,255,0.05)", border: "1px solid var(--card-border)", color: "#fff", marginTop: "20px" }}
          onClick={() => {
            setSuccessToken("");
            setName("");
            setEmail("");
          }}
        >
          Return to Store
        </button>
      </div>
    );
  }

  return (
    <div className="container">
      <h1>Payments</h1>
      <p className="subtitle">Secure digital downloads via Razorpay</p>

      {selectedProduct ? (
        <form onSubmit={handleCheckout}>
          <div className="product-card">
            <span style={{ fontSize: "12px", textTransform: "uppercase", color: "var(--accent-purple)", fontWeight: 600, letterSpacing: "1px" }}>
              Digital Product
            </span>
            <div className="product-title">{selectedProduct.name}</div>
            <div className="product-price">
              ₹{selectedProduct.price / 100}
              <span>INR</span>
            </div>
            <p style={{ fontSize: "13px", color: "var(--text-secondary)", marginTop: "4px" }}>
              Secure PDF download link will be instantly delivered to your email.
            </p>
          </div>

          <div className="form-group">
            <label htmlFor="name">Your Name</label>
            <input
              type="text"
              id="name"
              placeholder="e.g. John Doe"
              value={name}
              onChange={(e) => setName(e.target.value)}
              disabled={loading}
              required
            />
          </div>

          <div className="form-group">
            <label htmlFor="email">Email Address</label>
            <input
              type="email"
              id="email"
              placeholder="e.g. john@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              disabled={loading}
              required
            />
          </div>

          {error && <div className="error-message">{error}</div>}

          <button type="submit" className="btn-primary" disabled={loading}>
            {loading ? (
              <span style={{ display: "flex", alignItems: "center", justifyContent: "center", gap: "8px" }}>
                <span className="loader" style={{ width: "16px", height: "16px", borderWidth: "2px" }}></span>
                Processing Checkout...
              </span>
            ) : (
              `Pay ₹${selectedProduct.price / 100} & Download`
            )}
          </button>
        </form>
      ) : (
        <div className="error-message">No digital products found in the catalog.</div>
      )}
    </div>
  );
}

export default App;