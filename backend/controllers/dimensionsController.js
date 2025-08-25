const pool = require("../models/db");

exports.getDimensions = async (req, res) => {
  try {
    const result = await pool.query("SELECT * FROM dimensions ORDER BY id");
    res.json(result.rows);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
};

exports.createDimension = async (req, res) => {
  const { name, table_name, column_name, data_type, description } = req.body;
  try {
    const result = await pool.query(
      `INSERT INTO dimensions (name, table_name, column_name, data_type, description)
       VALUES ($1, $2, $3, $4, $5) RETURNING *`,
      [name, table_name, column_name, data_type, description]
    );
    res.status(201).json(result.rows[0]);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
};
