int expect(float a, float b) {
    if (!(a == b)) {
        printf("Failed\n");
        printf("  %d expected, but got %d\n", a, b);
        exit(1);
    }
}

int main() {
    printf("Testing float ... ");

    expect(1.0, 1.0);
    expect(1.5, 1.0 + 0.5);

    printf("OK\n");
    return 0;
}

int x() {
    expect(0.5, 1.0 - 0.5);
    expect(2.0, 1.0 * 2.0);
    expect(0.25, 1.0 / 4.0);

    expect(3.0, 1.0  + 2);
    expect(2.5, 5 - 2.5);
    expect(2.0, 1 * 2.0);
    expect(0.25, 1.0  / 4);
}

