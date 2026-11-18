// Prints N bytes from an address: ReadBytes.java ADDR N
import ghidra.app.script.GhidraScript;
public class ReadBytes extends GhidraScript {
  @Override public void run() throws Exception {
    var a = toAddr(getScriptArgs()[0]);
    int n = Integer.parseInt(getScriptArgs()[1]);
    StringBuilder sb = new StringBuilder();
    for (int i = 0; i < n; i++) sb.append(String.format("%02x ", currentProgram.getMemory().getByte(a.add(i)) & 0xff));
    println(sb.toString());
  }
}
