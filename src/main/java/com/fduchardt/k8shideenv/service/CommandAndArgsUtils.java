package com.fduchardt.k8shideenv.service;

import org.apache.logging.log4j.util.*;
import org.springframework.stereotype.*;

import java.util.*;
import java.util.stream.*;

@Component
public class CommandAndArgsUtils {

    public String reconcile(List<String> commands, List<String> args) {

        if (commands.isEmpty() && args.isEmpty()) {
            throw new RuntimeException("Either command or args have to be defined in K8s manifest");
        }

        List<String> allCommands = new ArrayList<>();
        allCommands.addAll(filterShell(commands));
        allCommands.addAll(filterShell(args));

        return String.join(" ", filterShell(allCommands.stream().filter(Strings::isNotBlank).collect(Collectors.toList())));
    }

    private List<String> filterShell(List<String> oldCommands) {
        if (oldCommands.size() > 1 && oldCommands.get(0).equals("sh") && oldCommands.get(1).equals("-c")) {
            return oldCommands.stream().skip(2).collect(Collectors.toList());
        } else {
            return new ArrayList<>(oldCommands);
        }
    }
}
