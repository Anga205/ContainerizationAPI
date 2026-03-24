import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;

public class Main {
    public static void main(String[] args) throws Exception {
        Path root = Path.of("/root");
        Files.createDirectories(root);
        Path p = root.resolve("test.txt");
        Files.writeString(p, "SecretData123", StandardCharsets.UTF_8);
        System.out.print(Files.readString(p, StandardCharsets.UTF_8));
    }
}
