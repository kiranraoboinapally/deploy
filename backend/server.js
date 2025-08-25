const express = require("express");
const cors = require("cors");
require("dotenv").config();

const factRoutes = require("./routes/factsRoutes");
const dimensionRoutes = require("./routes/dimensionsRoutes");
const cubeRoutes = require("./routes/cubesRoutes");

const app = express();
app.use(cors());
app.use(express.json());

app.use("/api/facts", factRoutes);
app.use("/api/dimensions", dimensionRoutes);
app.use("/api/cubes", cubeRoutes);

const PORT = process.env.PORT || 5000;
app.listen(PORT, () => console.log(`Server running on port ${PORT}`));
