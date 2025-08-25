const express = require("express");
const cors = require("cors");
require("dotenv").config();

const authRoutes = require("./routes/authRoutes");
const metadataRoutes = require("./routes/metadataRoutes");
const cubeRoutes = require("./routes/cubeRoutes");
const kpiRoutes = require("./routes/kpiRoutes");

const app = express();

app.use(cors());
app.use(express.json());

app.use("/api/auth", authRoutes);
app.use("/api/metadata", metadataRoutes);
app.use("/api/cubes", cubeRoutes);
app.use("/api/kpis", kpiRoutes);

const PORT = process.env.PORT || 5000;
app.listen(PORT, () => console.log(`Server running on port ${PORT}`));

app.use((req, res, next) => {
  console.log(`Incoming request: ${req.method} ${req.url}`);
  next();
});
