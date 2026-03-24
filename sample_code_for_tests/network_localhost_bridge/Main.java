import java.net.InetSocketAddress;
import java.net.Socket;

public class Main {
    public static void main(String[] args) throws Exception {
        try (Socket s = new Socket()) {
            s.connect(new InetSocketAddress("127.0.0.1", 8080), 1000);
            System.out.print("connected");
        } catch (Exception ex) {
            System.err.println(ex.getMessage());
            System.exit(1);
        }
    }
}
