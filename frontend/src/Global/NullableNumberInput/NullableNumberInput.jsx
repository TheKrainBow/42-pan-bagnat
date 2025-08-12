export default function NullableNumberInput({ value, onChange, placeholder }) {
  // Accepts number or "" (empty). Lets the user clear the field.
  const v = value === "" || value == null ? "" : String(value);
  return (
    <input
      className="rb-input"
      type="number"
      value={v}
      placeholder={placeholder}
      onChange={(e) => {
        const t = e.target.value;
        if (t === "") onChange("");
        else onChange(Number(t));
      }}
    />
  );
}