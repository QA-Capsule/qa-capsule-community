describe("Cypress sample suite", () => {
  it("passes quickly", () => {
    expect(200).to.equal(200);
  });

  it("intentional failure for demo", () => {
    expect(500).to.equal(200);
  });
});
