// routes/factsRoutes.js
const express = require("express");
const router = express.Router();
const authMiddleware = require("../middleware/authMiddleware");
const { permitRole } = require("../middleware/roleMiddleware");
const { getFacts, createFact } = require("../controllers/factsController");

router.get("/", authMiddleware, permitRole("user", "admin"), getFacts);
router.post("/", authMiddleware, permitRole("admin"), createFact);

module.exports = router;
