const pool = require("../models/db");

// GET all facts
exports.getFacts = async (req, res) => {
  try {
    const result = await pool.query("SELECT * FROM facts ORDER BY id");
    res.json(result.rows);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
};

// POST: Create a new fact
exports.createFact = async (req, res) => {
  const { name, table_name, column_name, aggregation_function, data_type, description } = req.body;
  try {
    const result = await pool.query(
      `INSERT INTO facts (name, table_name, column_name, aggregation_function, data_type, description)
       VALUES ($1, $2, $3, $4, $5, $6) RETURNING *`,
      [name, table_name, column_name, aggregation_function, data_type, description]
    );
    res.status(201).json(result.rows[0]);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
};
